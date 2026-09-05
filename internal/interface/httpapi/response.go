package httpapi

import (
	"encoding/json"
	"net/http"
)

type envelope struct {
	Data  any            `json:"data,omitempty"`
	Meta  any            `json:"meta,omitempty"`
	Error *errorEnvelope `json:"error,omitempty"`
}

type errorEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorBody is the shape every error response uses. Swagger references it.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps one error body — the standard failure envelope.
type ErrorResponse struct {
	Error *ErrorBody `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data})
}

func WriteMetaJSON(w http.ResponseWriter, status int, data, meta any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data, Meta: meta})
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Error: &errorEnvelope{Code: code, Message: message}})
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return false
	}
	return true
}
