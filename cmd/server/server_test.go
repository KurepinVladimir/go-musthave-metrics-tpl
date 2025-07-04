package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHandler_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
		check      func(t *testing.T, storage *MemStorage)
	}{
		{
			name:       "Valid Gauge",
			method:     http.MethodPost,
			url:        "/update/gauge/testGauge/42.5",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, storage *MemStorage) {
				val, ok := storage.gauges["testGauge"]
				assert.True(t, ok)
				assert.Equal(t, 42.5, val)
			},
		},
		{
			name:       "Valid Counter",
			method:     http.MethodPost,
			url:        "/update/counter/testCounter/5",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, storage *MemStorage) {
				val, ok := storage.counters["testCounter"]
				assert.True(t, ok)
				assert.Equal(t, int64(5), val)
			},
		},
		{
			name:       "Invalid Metric Type",
			method:     http.MethodPost,
			url:        "/update/unknown/test/123",
			wantStatus: http.StatusBadRequest,
			check:      func(t *testing.T, _ *MemStorage) {},
		},
		{
			name:       "Missing Metric Name",
			method:     http.MethodPost,
			url:        "/update/gauge//123",
			wantStatus: http.StatusNotFound,
			check:      func(t *testing.T, _ *MemStorage) {},
		},
		{
			name:       "Invalid Gauge Value",
			method:     http.MethodPost,
			url:        "/update/gauge/test/invalid",
			wantStatus: http.StatusBadRequest,
			check:      func(t *testing.T, _ *MemStorage) {},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage := NewMemStorage()
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
	storage := NewMemStorage()
	storage.UpdateGauge("myGauge", 42.5)
	storage.UpdateCounter("myCounter", 5)

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
	storage := NewMemStorage()
	storage.UpdateGauge("myGauge", 1.23)
	storage.UpdateCounter("myCounter", 99)

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
