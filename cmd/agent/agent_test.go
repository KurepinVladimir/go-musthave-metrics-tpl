package main

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"

	//"strconv"
	//"strings"
	"testing"
	//"bytes"
	"encoding/json"
	"io"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func collectRuntimeMetrics(metrics map[string]float64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics["Alloc"] = float64(m.Alloc)
	metrics["TotalAlloc"] = float64(m.TotalAlloc)
	metrics["NumGC"] = float64(m.NumGC)
	metrics["RandomValue"] = rand.Float64()
}

// func TestSendMetric(t *testing.T) {
// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != http.MethodPost {
// 			t.Errorf("Expected POST, got %s", r.Method)
// 		}
// 		if !strings.Contains(r.URL.Path, "/update/gauge/TestMetric/42.0") {
// 			t.Errorf("Unexpected URL path: %s", r.URL.Path)
// 		}
// 		w.WriteHeader(http.StatusOK)
// 	}))
// 	defer ts.Close()

// 	client := resty.New()
// 	serverURL := ts.URL + "/update"

// 	value := float64(42.0)
// 	metric := models.Metrics{
// 		ID:    "TestMetric",
// 		MType: "gauge",
// 		Value: &value,
// 	}
// 	_ = sendMetricJSON(client, serverURL, metric)

// 	// err := sendMetric(client, serverURL, "gauge", "TestMetric", "42.0")
// 	// if err != nil {
// 	// 	t.Errorf("sendMetric returned error: %v", err)
// 	// }

// 	err := sendMetricJSON(client, serverURL, metric)
// 	if err != nil {
// 		t.Errorf("sendMetric returned error: %v", err)
// 	}
// }

func TestSendMetricJSON(t *testing.T) {
	expectedMetric := models.Metrics{
		ID:    "TestMetric",
		MType: "gauge",
	}
	value := float64(42.0)
	expectedMetric.Value = &value

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/update", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		defer r.Body.Close()

		var received models.Metrics
		err = json.Unmarshal(body, &received)
		assert.NoError(t, err)

		assert.Equal(t, expectedMetric.ID, received.ID)
		assert.Equal(t, expectedMetric.MType, received.MType)
		assert.NotNil(t, received.Value)
		assert.Equal(t, *expectedMetric.Value, *received.Value)

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := resty.New()
	err := sendMetricJSON(client, ts.URL, expectedMetric)
	assert.NoError(t, err)
}

// func TestCollectRuntimeMetrics0(t *testing.T) {
// 	metrics := make(map[string]float64)
// 	collectRuntimeMetrics(metrics)

// 	expectedKeys := []string{"Alloc", "TotalAlloc", "RandomValue"}
// 	for _, key := range expectedKeys {
// 		if _, ok := metrics[key]; !ok {
// 			t.Errorf("Expected metric %s not found", key)
// 		}
// 	}
// 	if metrics["RandomValue"] < 0 || metrics["RandomValue"] > 1 {
// 		t.Errorf("RandomValue out of range: %f", metrics["RandomValue"])
// 	}
// 	if metrics["NumGC"] < 0 {
// 		t.Errorf("NumGC should not be negative: %s", strconv.FormatFloat(metrics["NumGC"], 'f', -1, 64))
// 	}
// }

func TestCollectRuntimeMetrics(t *testing.T) {
	metrics := make(map[string]float64)
	collectRuntimeMetrics(metrics)

	expectedKeys := []string{"Alloc", "TotalAlloc", "RandomValue", "NumGC"}
	for _, key := range expectedKeys {
		_, ok := metrics[key]
		assert.True(t, ok, "Expected metric %s not found", key)
	}

	assert.GreaterOrEqual(t, metrics["RandomValue"], 0.0)
	assert.LessOrEqual(t, metrics["RandomValue"], 1.0)
	assert.GreaterOrEqual(t, metrics["NumGC"], 0.0)
}
