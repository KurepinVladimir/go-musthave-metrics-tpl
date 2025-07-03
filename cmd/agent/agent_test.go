package main

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func collectRuntimeMetrics(metrics map[string]float64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics["Alloc"] = float64(m.Alloc)
	metrics["TotalAlloc"] = float64(m.TotalAlloc)
	metrics["NumGC"] = float64(m.NumGC)
	metrics["RandomValue"] = rand.Float64()
}

func TestSendMetric(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/update/gauge/TestMetric/42.0") {
			t.Errorf("Unexpected URL path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	serverURL := ts.URL + "/update"

	err := sendMetric(serverURL, "gauge", "TestMetric", "42.0")
	if err != nil {
		t.Errorf("sendMetric returned error: %v", err)
	}
}

func TestCollectRuntimeMetrics(t *testing.T) {
	metrics := make(map[string]float64)
	collectRuntimeMetrics(metrics)

	expectedKeys := []string{"Alloc", "TotalAlloc", "RandomValue"}
	for _, key := range expectedKeys {
		if _, ok := metrics[key]; !ok {
			t.Errorf("Expected metric %s not found", key)
		}
	}
	if metrics["RandomValue"] < 0 || metrics["RandomValue"] > 1 {
		t.Errorf("RandomValue out of range: %f", metrics["RandomValue"])
	}
	if metrics["NumGC"] < 0 {
		t.Errorf("NumGC should not be negative: %s", strconv.FormatFloat(metrics["NumGC"], 'f', -1, 64))
	}
}
