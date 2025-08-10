package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	writer       *gzip.Writer
	wroteHeaders bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if !w.wroteHeaders {
		w.Header().Set("Content-Encoding", "gzip")
		w.wroteHeaders = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeaders {
		w.WriteHeader(http.StatusOK) // установить код по умолчанию
	}
	return w.writer.Write(b)
}

func gzipResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем поддержку gzip и нужный тип ответа
		accepts := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		if !accepts {
			next.ServeHTTP(w, r)
			return
		}

		// Используем ResponseWriter с gzip
		w.Header().Add("Vary", "Accept-Encoding")

		gzw := gzip.NewWriter(w)
		defer gzw.Close()

		grw := &gzipResponseWriter{
			ResponseWriter: w,
			writer:         gzw,
		}

		next.ServeHTTP(grw, r)
	})
}

// middleware для чтения gzip-запросов
func gzipRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "failed to read gzip body", http.StatusBadRequest)
				return
			}
			defer gr.Close()
			r.Body = io.NopCloser(gr)
		}
		next.ServeHTTP(w, r)
	})
}
