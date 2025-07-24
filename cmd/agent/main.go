package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"

	//"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/logger"
	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var pollCount int64     // Счётчик обновлений метрик
var randomValue float64 // Значение случайной метрики

// отправка метрики на сервер
func sendMetricJSON(client *resty.Client, serverURL string, metric models.Metrics) error {

	// Сериализуем метрику в JSON
	var jsonBuf bytes.Buffer
	if err := json.NewEncoder(&jsonBuf).Encode(metric); err != nil {
		logger.Log.Debug("json encode error:", zap.Error(err))
		return err
	}

	// Сжимаем JSON в gzip
	var gzBuf bytes.Buffer
	gz := gzip.NewWriter(&gzBuf)
	if _, err := gz.Write(jsonBuf.Bytes()); err != nil {
		logger.Log.Debug("gzip write error:", zap.Error(err))
		return err
	}
	if err := gz.Close(); err != nil {
		logger.Log.Debug("gzip close error:", zap.Error(err))
		return err
	}

	// Отправляем сжатый JSON
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip"). // Говорим серверу: "Я поддерживаю сжатые ответы"
		SetBody(gzBuf.Bytes()).
		Post(serverURL + "/update")
	if err != nil {
		logger.Log.Debug("send error:", zap.Error(err))
		return err
	}

	if resp.IsError() {
		logger.Log.Debug("server returned error", zap.Int("status", resp.StatusCode()), zap.String("body", resp.String()))
		return err
	}
	logger.Log.Debug("metric sent", zap.String("id", metric.ID), zap.String("type", metric.MType))
	return nil

}

func main() {
	// обрабатываем аргументы командной строки
	parseFlags()
	run()
}

func run() {

	reportInterval := time.Duration(flagReportInterval) * time.Second // Интервал отправки метрик на сервер, по умолчанию 10 секунд
	pollInterval := time.Duration(flagPollInterval) * time.Second     // Интервал обновления метрик, по умолчанию 2 секунды

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

			// // Обновляем RandomValue
			randomValue = rand.Float64()

			// // Увеличиваем счётчик обновлений
			pollCount++

			//logger.Log.Debug("collected metrics", zap.Int("count", len(runtimeMetrics)))

		case <-tickerReport.C:

			// Отправка runtime метрик
			for name, val := range runtimeMetrics {
				value := val
				metric := models.Metrics{
					ID:    name,
					MType: "gauge",
					Value: &value,
				}
				_ = sendMetricJSON(client, flagRunAddr, metric)
			}

			// Отправка RandomValue
			value := randomValue
			_ = sendMetricJSON(client, flagRunAddr, models.Metrics{
				ID:    "RandomValue",
				MType: "gauge",
				Value: &value,
			})
			//logger.Log.Debug("metrics reported", zap.String("RandomValue", fmt.Sprintf("%f", randomValue)))

			// Отправка PollCount
			delta := pollCount
			_ = sendMetricJSON(client, flagRunAddr, models.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: &delta,
			})
			//logger.Log.Debug("metrics reported", zap.Int64("PollCount", pollCount))
			logger.Log.Debug("metrics reported")
		}

	}
}
