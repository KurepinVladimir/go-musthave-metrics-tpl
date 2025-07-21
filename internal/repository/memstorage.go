package repository

import (
	"context"
	"sync"
)

// Storage описывает поведение хранилища метрик
type Storage interface {
	UpdateGauge(ctx context.Context, name string, value float64)
	UpdateCounter(ctx context.Context, name string, value int64)
	GetGauge(ctx context.Context, name string) (float64, bool)
	GetCounter(ctx context.Context, name string) (int64, bool)
	GetAllMetrics(ctx context.Context) (map[string]float64, map[string]int64)
}

// MemStorage реализует интерфейс Storage. хранилища в памяти
type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage создаёт новое хранилище
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge устанавливает значение метрики типа gauge
func (s *MemStorage) UpdateGauge(_ context.Context, name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

// UpdateCounter увеличивает значение метрики типа counter
func (s *MemStorage) UpdateCounter(_ context.Context, name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
}

func (s *MemStorage) GetGauge(_ context.Context, name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.gauges[name]
	return val, ok
}

func (s *MemStorage) GetCounter(_ context.Context, name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.counters[name]
	return val, ok
}

func (s *MemStorage) GetAllMetrics(_ context.Context) (map[string]float64, map[string]int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// создаём копии, чтобы не отдавать оригинальные мапы
	gaugeCopy := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		gaugeCopy[k] = v
	}
	counterCopy := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		counterCopy[k] = v
	}
	return gaugeCopy, counterCopy
}
