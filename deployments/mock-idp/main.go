// Command mock-idp is a tiny, dependency-free OIDC identity provider used to
// exercise the Entra login flow locally (and in CI) without Azure. It is NOT
// for production: it implements just the discovery / authorize / token /
// jwks endpoints the go-oidc client needs, signing id_tokens with a throwaway
// RSA key minted at startup.
package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	clientID     = "mock-client"
	clientSecret = "mock-secret"
	issuer       = "http://mock-idp:9090"
	kid          = "mock-rsa-1"
)

type code struct {
	Sub, Email, Name string
	ExpiresAt        time.Time
}

type server struct {
	key    *rsa.PrivateKey
	jwks   []byte
	config []byte
	mu     sync.Mutex
	codes  map[string]code
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{key: key, codes: make(map[string]code)}
	s.jwks = s.buildJWKS()
	s.config = s.buildConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", s.handleDiscovery)
	mux.HandleFunc("/jwks", s.handleJWKS)
	mux.HandleFunc("/authorize", s.handleAuthorize)
	mux.HandleFunc("/token", s.handleToken)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("mock-idp listening on :%s (issuer %s)", port, issuer)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(s.config)
}

func (s *server) handleJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(s.jwks)
}

// handleAuthorize performs the browser leg: validate params, mint a fresh
// single-use code, then bounce the browser back to the registered redirect_uri.
func (s *server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("response_type") != "code" || q.Get("redirect_uri") == "" {
		http.Error(w, "unsupported or missing oauth parameters", http.StatusBadRequest)
		return
	}
	email := q.Get("login_hint")
	if email == "" {
		email = "sso-user@example.com"
	}
	sub := "mock-sub-" + hex.EncodeToString([]byte(email))
	if len(sub) > 40 {
		sub = sub[:40]
	}

	s.mu.Lock()
	c := randomHex(12)
	s.codes[c] = code{
		Sub:       sub,
		Email:     email,
		Name:      "SSO Demo User",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

	loc := fmt.Sprintf("%s?code=%s&state=%s", q.Get("redirect_uri"), c, q.Get("state"))
	http.Redirect(w, r, loc, http.StatusFound)
}

// handleToken is the server-to-server leg: validate the client's basic
// credentials, consume the code and return access_token + id_token.
func (s *server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	id, secret := r.Form.Get("client_id"), r.Form.Get("client_secret")
	if b, bpw, ok := r.BasicAuth(); ok {
		id, secret = b, bpw
	}
	if id != clientID || secret != clientSecret {
		http.Error(w, "invalid client credentials", http.StatusUnauthorized)
		return
	}
	if r.Form.Get("grant_type") != "authorization_code" {
		http.Error(w, "unsupported grant_type", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	cd, ok := s.codes[r.Form.Get("code")]
	delete(s.codes, r.Form.Get("code"))
	s.mu.Unlock()
	if !ok || time.Now().After(cd.ExpiresAt) {
		http.Error(w, "invalid or expired code", http.StatusBadRequest)
		return
	}

	now := time.Now()
	idToken, err := s.signJWT(map[string]any{
		"iss":               issuer,
		"sub":               cd.Sub,
		"aud":               clientID,
		"iat":               now.Unix(),
		"exp":               now.Add(time.Hour).Unix(),
		"email":             cd.Email,
		"email_verified":    true,
		"name":              cd.Name,
		"preferred_username": cd.Email,
	})
	if err != nil {
		http.Error(w, "token signing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": idToken,
		"id_token":     idToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

func (s *server) buildConfig() []byte {
	cfg := map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/authorize",
		"token_endpoint":                        issuer + "/token",
		"jwks_uri":                              issuer + "/jwks",
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
	}
	b, _ := json.Marshal(cfg)
	return b
}

func (s *server) buildJWKS() []byte {
	pub := s.key.Public().(*rsa.PublicKey)
	key := map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kid,
		"n":   b64(pub.N.Bytes()),
		"e":   b64([]byte{0x01, 0x00, 0x01}),
	}
	b, _ := json.Marshal(map[string]any{"keys": []map[string]any{key}})
	return b
}

// signJWT produces an RS256 JWS: header.payload signed with SHA-256 PKCS#1 v1.5.
func (s *server) signJWT(claims map[string]any) (string, error) {
	hb, _ := json.Marshal(map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid})
	cb, _ := json.Marshal(claims)

	signingInput := b64(hb) + "." + b64(cb)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + b64(sig), nil
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}