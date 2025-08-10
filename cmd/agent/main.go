package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"math/rand"
	"runtime"
	"time"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/logger"
	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// Agent инкапсулирует состояние и поведение агента для сбора и отправки метрик на сервер
type Agent struct {
	PollCount   int64              // счётчик обновлений метрик
	RandomValue float64            // случайное значение метрики
	Metrics     map[string]float64 // метрики типа gauge из runtime
	Client      *resty.Client      // HTTP-клиент
	ServerURL   string             // адрес сервера
}

// NewAgent создаёт и возвращает новый экземпляр агента
func NewAgent(serverURL string) *Agent {
	return &Agent{
		Metrics:   make(map[string]float64), // инициализируем хранилище метрик
		Client:    resty.New(),              // Создаём HTTP-клиент resty
		ServerURL: serverURL,                // Адрес сервера, куда будем отправлять метрики
	}
}

// sendMetricJSON отправляет одну метрику на сервер в формате JSON, сжатом через gzip
func (a *Agent) sendMetricJSON(metric models.Metrics) error {

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
	resp, err := a.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip"). // Говорим серверу: "Я поддерживаю сжатые ответы"
		SetBody(gzBuf.Bytes()).
		Post(a.ServerURL + "/update")
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

// collectMetrics собирает метрики из runtime и обновляет состояние агента
func (a *Agent) collectMetrics() {

	var m runtime.MemStats   // Считываем текущие значения метрик
	runtime.ReadMemStats(&m) // Обновляем метрики в агенте

	// обновляем runtime метрики
	a.Metrics["Alloc"] = float64(m.Alloc)
	a.Metrics["BuckHashSys"] = float64(m.BuckHashSys)
	a.Metrics["Frees"] = float64(m.Frees)
	a.Metrics["GCCPUFraction"] = m.GCCPUFraction
	a.Metrics["GCSys"] = float64(m.GCSys)
	a.Metrics["HeapAlloc"] = float64(m.HeapAlloc)
	a.Metrics["HeapIdle"] = float64(m.HeapIdle)
	a.Metrics["HeapInuse"] = float64(m.HeapInuse)
	a.Metrics["HeapObjects"] = float64(m.HeapObjects)
	a.Metrics["HeapReleased"] = float64(m.HeapReleased)
	a.Metrics["HeapSys"] = float64(m.HeapSys)
	a.Metrics["LastGC"] = float64(m.LastGC)
	a.Metrics["Lookups"] = float64(m.Lookups)
	a.Metrics["MCacheInuse"] = float64(m.MCacheInuse)
	a.Metrics["MCacheSys"] = float64(m.MCacheSys)
	a.Metrics["MSpanInuse"] = float64(m.MSpanInuse)
	a.Metrics["MSpanSys"] = float64(m.MSpanSys)
	a.Metrics["Mallocs"] = float64(m.Mallocs)
	a.Metrics["NextGC"] = float64(m.NextGC)
	a.Metrics["NumForcedGC"] = float64(m.NumForcedGC)
	a.Metrics["NumGC"] = float64(m.NumGC)
	a.Metrics["OtherSys"] = float64(m.OtherSys)
	a.Metrics["PauseTotalNs"] = float64(m.PauseTotalNs)
	a.Metrics["StackInuse"] = float64(m.StackInuse)
	a.Metrics["StackSys"] = float64(m.StackSys)
	a.Metrics["Sys"] = float64(m.Sys)
	a.Metrics["TotalAlloc"] = float64(m.TotalAlloc)

	a.RandomValue = rand.Float64() // Обновляем случайное значение метрики
	a.PollCount++                  // Увеличиваем счётчик обновлений
}

// reportMetrics отправляет все собранные метрики на сервер
func (a *Agent) reportMetrics() {

	for name, val := range a.Metrics {
		value := val
		metric := models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &value,
		}
		_ = a.sendMetricJSON(metric) // намеренно игнорируем ошибку, но логируем
	}

	random := a.RandomValue
	_ = a.sendMetricJSON(models.Metrics{ // намеренно игнорируем ошибку, но логируем
		ID:    "RandomValue",
		MType: "gauge",
		Value: &random,
	})

	poll := a.PollCount
	_ = a.sendMetricJSON(models.Metrics{ // намеренно игнорируем ошибку, но логируем
		ID:    "PollCount",
		MType: "counter",
		Delta: &poll,
	})

	logger.Log.Debug("metrics reported")
}

func main() {

	parseFlags() // обрабатываем аргументы командной строки

	// запускаем агента
	reportInterval := time.Duration(flagReportInterval) * time.Second // Интервал отправки метрик на сервер, по умолчанию 10 секунд
	pollInterval := time.Duration(flagPollInterval) * time.Second     // Интервал обновления метрик, по умолчанию 2 секунды

	agent := NewAgent(flagRunAddr)                 // Создаём нового агента с адресом сервера
	tickerPoll := time.NewTicker(pollInterval)     // Таймер для обновления метрик
	tickerReport := time.NewTicker(reportInterval) // Таймер для отправки метрик
	defer tickerPoll.Stop()
	defer tickerReport.Stop()

	for {
		select {
		case <-tickerPoll.C:
			agent.collectMetrics() // Сбор метрик из runtime и обновление состояния агента
		case <-tickerReport.C:
			agent.reportMetrics() // Отправка собранных метрик на сервер
		}
	}
}
