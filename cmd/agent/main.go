package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	reportInterval = 10 * time.Second // Интервал отправки метрик на сервер
	pollInterval   = 2 * time.Second  // Интервал обновления метрик
)

var pollCount int64 // Счётчик обновлений метрик

// отправка метрики на сервер
func sendMetric(client *resty.Client, serverURL, metricType, name, value string) error {

	url := fmt.Sprintf("%s/%s/%s/%s", serverURL, metricType, name, value)

	resp, err := client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url) // Отправка POST-запроса
	if err != nil {
		fmt.Println("send error:", err)
		return err
	}

	if resp.IsError() {
		fmt.Printf("server returned error: %s\n", resp.Status())
	}
	return nil

}

func main() {

	serverURL := "http://localhost:8080/update"

	// Создаём HTTP-клиент resty
	client := resty.New()

	// Хранилище runtime метрик
	runtimeMetrics := make(map[string]float64)

	// Таймеры для обновления и отправки метрик
	tickerPoll := time.NewTicker(pollInterval)
	tickerReport := time.NewTicker(reportInterval)
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	for {
		select {
		case <-tickerPoll.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m) // Считываем текущие значения метрик

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

			// Увеличиваем счётчик обновлений
			pollCount++

		case <-tickerReport.C:
			// Отправляем все метрики типа gauge
			for name, value := range runtimeMetrics {
				err := sendMetric(client, serverURL, "gauge", name, strconv.FormatFloat(value, 'f', -1, 64))
				if err != nil {
					fmt.Println("sendMetric returned error:", err)
				}
			}

			// Отправляем метрику PollCount
			err := sendMetric(client, serverURL, "counter", "PollCount", strconv.FormatInt(pollCount, 10))
			if err != nil {
				fmt.Println("sendMetric returned error:", err)
			}
		}
	}
}
