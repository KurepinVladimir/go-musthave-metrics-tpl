package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Тип хранилища

// MemStorage реализует интерфейс Storage
type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// Storage описывает поведение хранилища метрик
type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
}

// NewMemStorage создаёт новое хранилище
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge устанавливает значение метрики типа gauge
func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

// UpdateCounter увеличивает значение метрики типа counter
func (s *MemStorage) UpdateCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
}

// handler обрабатывает POST-запросы на /update/<тип>/<имя>/<значение>
func handler(storage Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/update/"), "/")
		if len(parts) != 3 {
			http.Error(w, "Incorrect URL format", http.StatusNotFound)
			return
		}

		metricType := parts[0]
		name := parts[1]
		valueStr := parts[2]

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
			storage.UpdateGauge(name, value)

		case "counter":
			value, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				http.Error(w, "Invalid counter value", http.StatusBadRequest)
				return
			}
			storage.UpdateCounter(name, value)

		default:
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}
}

func main() {
	storage := NewMemStorage()
	http.HandleFunc("/update/", handler(storage))
	log.Println("Starting server at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
