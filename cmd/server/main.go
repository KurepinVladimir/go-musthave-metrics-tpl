package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/logger"
	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// handler обрабатывает POST-запросы на /update/{type}/{name}/{value}
func updateHandler(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		metricType := chi.URLParam(r, "type")
		name := chi.URLParam(r, "name")
		valueStr := chi.URLParam(r, "value")

		if name == "" {
			http.Error(w, "Missing metric name", http.StatusNotFound)
			return
		}

		switch metricType {
		case "gauge":
			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				http.Error(w, "Invalid gauge value", http.StatusBadRequest)
				return
			}
			storage.UpdateGauge(r.Context(), name, value)

		case "counter":
			value, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				http.Error(w, "Invalid counter value", http.StatusBadRequest)
				return
			}
			storage.UpdateCounter(r.Context(), name, value)

		default:
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}
}

func updateHandlerJSON(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// десериализуем запрос в структуру модели
		logger.Log.Debug("decoding request")
		var m models.Metrics
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&m); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		switch m.MType {
		case "gauge":
			if m.Value == nil {
				http.Error(w, "missing gauge value", http.StatusBadRequest)
				return
			}
			storage.UpdateGauge(r.Context(), m.ID, *m.Value)
		case "counter":
			if m.Delta == nil {
				http.Error(w, "missing counter delta", http.StatusBadRequest)
				return
			}
			storage.UpdateCounter(r.Context(), m.ID, *m.Delta)
		default:
			http.Error(w, "unknown metric type", http.StatusNotImplemented)
			return
		}

		// w.Header().Set("Content-Type", "application/json")
		// w.WriteHeader(http.StatusOK)
		// json.NewEncoder(w).Encode(m)

		// установим правильный заголовок для типа данных
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// сериализуем ответ сервера
		enc := json.NewEncoder(w)
		if err := enc.Encode(m); err != nil {
			logger.Log.Debug("error encoding response", zap.Error(err))
			return
		}
		logger.Log.Debug("sending HTTP 200 response")
	}
}

func valueHandlerJSON(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch m.MType {
		case "gauge":
			val, ok := storage.GetGauge(r.Context(), m.ID)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			m.Value = &val
		case "counter":
			val, ok := storage.GetCounter(r.Context(), m.ID)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			m.Delta = &val
		default:
			http.Error(w, "unknown metric type", http.StatusNotImplemented)
			return
		}
		json.NewEncoder(w).Encode(m)
	}
}

// GET /value/{type}/{name}
func valueHandler(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		name := chi.URLParam(r, "name")

		switch metricType {
		case "gauge":
			val, ok := storage.GetGauge(r.Context(), name)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, strconv.FormatFloat(val, 'f', -1, 64))

		case "counter":
			val, ok := storage.GetCounter(r.Context(), name)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "%d", val)

		default:
			http.Error(w, "invalid metric type", http.StatusBadRequest)
		}
	}
}

// GET /
func indexHandler(storage repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gauges, counters := storage.GetAllMetrics(r.Context())

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintln(w, "<html><body><h1>Metrics</h1><ul>")
		for name, val := range gauges {
			fmt.Fprintf(w, "<li>gauge %s = %f</li>\n", name, val)
		}
		for name, val := range counters {
			fmt.Fprintf(w, "<li>counter %s = %d</li>\n", name, val)
		}
		fmt.Fprintln(w, "</ul></body></html>")
	}
}

func main() {

	// обрабатываем аргументы командной строки
	parseFlags()

	if err := run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() error {

	if err := logger.Initialize("INFO"); err != nil {
		return err
	}

	storage := repository.NewMemStorage()

	r := chi.NewRouter()

	//Use добавляет middleware ко всем маршрутам, зарегистрированным через chi.Router.
	r.Use(logger.RequestLogger)
	// Добавляем middleware для обработки gzip-запросов и ответов
	r.Use(gzipRequestMiddleware)
	r.Use(gzipResponseMiddleware)

	r.Post("/update/{type}/{name}/{value}", updateHandler(storage)) // Регистрируем маршрут с параметрами

	r.Post("/update", updateHandlerJSON(storage))
	r.Post("/update/", updateHandlerJSON(storage))

	r.Post("/value", valueHandlerJSON(storage))
	r.Post("/value/", valueHandlerJSON(storage))

	r.Get("/value/{type}/{name}", valueHandler(storage))
	r.Get("/", indexHandler(storage))

	//log.Println("Starting server at", flagRunAddr)
	logger.Log.Info("Running server", zap.String("address", flagRunAddr))

	return http.ListenAndServe(flagRunAddr, r)
	//return http.ListenAndServe(flagRunAddr, logger.RequestLogger(r))
}
