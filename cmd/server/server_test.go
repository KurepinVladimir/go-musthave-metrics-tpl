package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHandler_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
		check      func(t *testing.T, storage *repository.MemStorage)
	}{
		{
			name:       "Valid Gauge",
			method:     http.MethodPost,
			url:        "/update/gauge/testGauge/42.5",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, storage *repository.MemStorage) {
				val, ok := storage.GetGauge(context.Background(), "testGauge")
				assert.True(t, ok)
				assert.Equal(t, 42.5, val)
			},
		},
		{
			name:       "Valid Counter",
			method:     http.MethodPost,
			url:        "/update/counter/testCounter/5",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, storage *repository.MemStorage) {
				val, ok := storage.GetCounter(context.Background(), "testCounter")
				assert.True(t, ok)
				assert.Equal(t, int64(5), val)
			},
		},
		{
			name:       "Invalid Metric Type",
			method:     http.MethodPost,
			url:        "/update/unknown/test/123",
			wantStatus: http.StatusBadRequest,
			check:      func(t *testing.T, _ *repository.MemStorage) {},
		},
		{
			name:       "Missing Metric Name",
			method:     http.MethodPost,
			url:        "/update/gauge//123",
			wantStatus: http.StatusNotFound,
			check:      func(t *testing.T, _ *repository.MemStorage) {},
		},
		{
			name:       "Invalid Gauge Value",
			method:     http.MethodPost,
			url:        "/update/gauge/test/invalid",
			wantStatus: http.StatusBadRequest,
			check:      func(t *testing.T, _ *repository.MemStorage) {},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", updateHandler(storage))

			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tc.wantStatus, resp.StatusCode)
			tc.check(t, storage)
		})
	}
}

func TestGetValueHandler(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.UpdateGauge(context.Background(), "myGauge", 42.5)
	storage.UpdateCounter(context.Background(), "myCounter", 5)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", valueHandler(storage))

	tests := []struct {
		name       string
		url        string
		expected   string
		statusCode int
	}{
		{"Existing Gauge", "/value/gauge/myGauge", "42.5", http.StatusOK},
		{"Existing Counter", "/value/counter/myCounter", "5", http.StatusOK},
		{"Unknown Gauge", "/value/gauge/unknown", "", http.StatusNotFound},
		{"Unknown Counter", "/value/counter/unknown", "", http.StatusNotFound},
		{"Invalid Type", "/value/unknown/type", "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)

			assert.Equal(t, tt.statusCode, res.StatusCode)
			if res.StatusCode == http.StatusOK {
				assert.Equal(t, tt.expected, string(body))
			}
		})
	}
}

func TestHTMLHandler(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.UpdateGauge(context.Background(), "myGauge", 1.23)
	storage.UpdateCounter(context.Background(), "myCounter", 99)

	r := chi.NewRouter()
	r.Get("/", indexHandler(storage))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "myGauge")
	assert.Contains(t, string(body), "1.230")
	assert.Contains(t, string(body), "myCounter")
	assert.Contains(t, string(body), "99")
}

func TestUpdateHandlerJSON(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := updateHandlerJSON(storage)

	tests := []struct {
		name       string
		input      string
		wantStatus int
		check      func() error
	}{
		{
			name:       "valid gauge metric",
			input:      `{"id":"TestGauge","type":"gauge","value":123.456}`,
			wantStatus: http.StatusOK,
			check: func() error {
				v, ok := storage.GetGauge(context.Background(), "TestGauge")
				if !ok || v != 123.456 {
					return fmt.Errorf("expected 123.456, got %v (ok=%v)", v, ok)
				}
				return nil
			},
		},
		{
			name:       "valid counter metric",
			input:      `{"id":"TestCounter","type":"counter","delta":5}`,
			wantStatus: http.StatusOK,
			check: func() error {
				v, ok := storage.GetCounter(context.Background(), "TestCounter")
				if !ok || v != 5 {
					return fmt.Errorf("expected 5, got %v (ok=%v)", v, ok)
				}
				return nil
			},
		},
		{
			name:       "missing value",
			input:      `{"id":"NoValue","type":"gauge"}`,
			wantStatus: http.StatusBadRequest,
			check:      func() error { return nil },
		},
		{
			name:       "unknown type",
			input:      `{"id":"BadType","type":"other","value":1.23}`,
			wantStatus: http.StatusNotImplemented,
			check:      func() error { return nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler(rr, req)
			res := rr.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("got status %d, want %d", res.StatusCode, tt.wantStatus)
			}
			if err := tt.check(); err != nil {
				t.Errorf("check failed: %v", err)
			}
		})
	}
}

func TestValueHandlerJSON(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.UpdateGauge(context.Background(), "G1", 99.9)
	storage.UpdateCounter(context.Background(), "C1", 7)

	handler := valueHandlerJSON(storage)

	tests := []struct {
		name       string
		input      string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "existing gauge",
			input:      `{"id":"G1","type":"gauge"}`,
			wantStatus: http.StatusOK,
			wantBody:   `"value":99.9`,
		},
		{
			name:       "existing counter",
			input:      `{"id":"C1","type":"counter"}`,
			wantStatus: http.StatusOK,
			wantBody:   `"delta":7`,
		},
		{
			name:       "not found",
			input:      `{"id":"none","type":"gauge"}`,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler(rr, req)
			res := rr.Result()
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body)

			if res.StatusCode != tt.wantStatus {
				t.Errorf("got status %d, want %d", res.StatusCode, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(string(body), tt.wantBody) {
				t.Errorf("expected body to contain %q, got %s", tt.wantBody, body)
			}
		})
	}
}

// Агент может отправить gzip-запрос
func TestUpdateHandlerJSON_GzipRequest(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := updateHandlerJSON(storage)

	// JSON-метрика
	input := `{"id":"GZGauge","type":"gauge","value":3.14}`

	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(input))
	assert.NoError(t, err)
	assert.NoError(t, gz.Close())

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(buf.String()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	// Оборачиваем хендлер миддлварой
	wrapped := gzipRequestMiddleware(handler)
	wrapped.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	v, ok := storage.GetGauge(context.Background(), "GZGauge")
	assert.True(t, ok)
	assert.Equal(t, 3.14, v)
}

// Сервер сжимает ответ, если клиент просит gzip
func TestValueHandlerJSON_GzipResponse(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.UpdateGauge(context.Background(), "GZGauge", 2.718)

	handler := valueHandlerJSON(storage)

	// JSON-запрос
	body := `{"id":"GZGauge","type":"gauge"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()

	// Оборачиваем хендлер миддлварой
	wrapped := gzipResponseMiddleware(handler)
	wrapped.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))

	// Распаковываем ответ
	gr, err := gzip.NewReader(res.Body)
	assert.NoError(t, err)
	defer gr.Close()

	uncompressed, err := io.ReadAll(gr)
	assert.NoError(t, err)

	assert.Contains(t, string(uncompressed), `"value":2.718`)
}
