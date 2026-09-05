package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

const requestIDHeader = "X-Request-ID"

// RequestID assigns (or forwards) a unique id per request so logs and audit
// entries can be correlated end-to-end.
func RequestID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(requestIDHeader, id)
		h.ServeHTTP(w, r.WithContext(httpctx.WithRequestID(r.Context(), id)))
	})
}

// ResponseWriter tracks status and latency for structured logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
	start  time.Time
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}
