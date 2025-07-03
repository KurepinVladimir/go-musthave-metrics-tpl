package main

import (
	"reflect"
	"testing"

	"net/http"
	"net/http/httptest"
)

func TestHandler(t *testing.T) {
	type args struct {
		storage Storage
	}
	tests := []struct {
		name string
		args args
		want http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler(tt.args.storage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidGaugeUpdate(t *testing.T) {
	storage := NewMemStorage()
	h := handler(storage)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/testMetric/42.5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	val, ok := storage.gauges["testMetric"]
	if !ok || val != 42.5 {
		t.Errorf("Expected gauge testMetric=42.5, got %f", val)
	}
}

func TestValidCounterUpdate(t *testing.T) {
	storage := NewMemStorage()
	h := handler(storage)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/testCounter/5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	val, ok := storage.counters["testCounter"]
	if !ok || val != 5 {
		t.Errorf("Expected counter testCounter=5, got %d", val)
	}
}

func TestInvalidMetricType(t *testing.T) {
	storage := NewMemStorage()
	h := handler(storage)

	req := httptest.NewRequest(http.MethodPost, "/update/unknown/test/123", nil)
	w := httptest.NewRecorder()

	h(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestMissingMetricName(t *testing.T) {
	storage := NewMemStorage()
	h := handler(storage)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge//123", nil)
	w := httptest.NewRecorder()

	h(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 Not Found, got %d", resp.StatusCode)
	}
}

func TestInvalidValue(t *testing.T) {
	storage := NewMemStorage()
	h := handler(storage)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/invalid", nil)
	w := httptest.NewRecorder()

	h(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request, got %d", resp.StatusCode)
	}
}
