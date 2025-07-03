package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

const (
	reportInterval = 10 * time.Second
	pollInterval   = 2 * time.Second
)

var pollCount int64

// отправка метрики на сервер
func sendMetric(serverURL, metricType, name, value string) error {

	url := fmt.Sprintf("%s/%s/%s/%s", serverURL, metricType, name, value)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Println("create request error:", err)
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("send error:", err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

func main() {

	serverURL := "http://localhost:8080/update"

	runtimeMetrics := make(map[string]float64)

	tickerPoll := time.NewTicker(pollInterval)
	tickerReport := time.NewTicker(reportInterval)
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	for {
		select {
		case <-tickerPoll.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// обновляем runtime метрики
			runtimeMetrics["Alloc"] = float64(m.Alloc)
			runtimeMetrics["BuckHashSys"] = float64(m.BuckHashSys)
			runtimeMetrics["Frees"] = float64(m.Frees)
			runtimeMetrics["GCCPUFraction"] = m.GCCPUFraction
			runtimeMetrics["GCSys"] = float64(m.GCSys)
			runtimeMetrics["HeapAlloc"] = float64(m.HeapAlloc)
			runtimeMetrics["HeapIdle"] = float64(m.HeapIdle)
			runtimeMetrics["HeapInuse"] = float64(m.HeapInuse)
			runtimeMetrics["HeapObjects"] = float64(m.HeapObjects)
			runtimeMetrics["HeapReleased"] = float64(m.HeapReleased)
			runtimeMetrics["HeapSys"] = float64(m.HeapSys)
			runtimeMetrics["LastGC"] = float64(m.LastGC)
			runtimeMetrics["Lookups"] = float64(m.Lookups)
			runtimeMetrics["MCacheInuse"] = float64(m.MCacheInuse)
			runtimeMetrics["MCacheSys"] = float64(m.MCacheSys)
			runtimeMetrics["MSpanInuse"] = float64(m.MSpanInuse)
			runtimeMetrics["MSpanSys"] = float64(m.MSpanSys)
			runtimeMetrics["Mallocs"] = float64(m.Mallocs)
			runtimeMetrics["NextGC"] = float64(m.NextGC)
			runtimeMetrics["NumForcedGC"] = float64(m.NumForcedGC)
			runtimeMetrics["NumGC"] = float64(m.NumGC)
			runtimeMetrics["OtherSys"] = float64(m.OtherSys)
			runtimeMetrics["PauseTotalNs"] = float64(m.PauseTotalNs)
			runtimeMetrics["StackInuse"] = float64(m.StackInuse)
			runtimeMetrics["StackSys"] = float64(m.StackSys)
			runtimeMetrics["Sys"] = float64(m.Sys)
			runtimeMetrics["TotalAlloc"] = float64(m.TotalAlloc)

			// RandomValue
			runtimeMetrics["RandomValue"] = rand.Float64()

			pollCount++

		case <-tickerReport.C:
			for name, value := range runtimeMetrics {
				err := sendMetric(serverURL, "gauge", name, strconv.FormatFloat(value, 'f', -1, 64))

				if err != nil {
					fmt.Println("sendMetric returned error:", err)
				}

			}

			err := sendMetric(serverURL, "counter", "PollCount", strconv.FormatInt(pollCount, 10))
			if err != nil {
				fmt.Println("sendMetric returned error:", err)
			}
		}
	}
}
