package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/cryptohelpers"
)

func ValidateHashSHA256(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			sentHash := r.Header.Get("HashSHA256")
			if sentHash == "" {
				http.Error(w, "missing HashSHA256 header", http.StatusBadRequest)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "unable to read body", http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if !cryptohelpers.Compare(bodyBytes, key, sentHash) {
				http.Error(w, "invalid signature", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
