// cmd/agent/batcher.go
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"time"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/logger"
	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Batcher struct {
	in       chan models.Metrics
	flushInt time.Duration
	maxSize  int
	client   *resty.Client
	endpoint string
}

func NewBatcher(endpoint string, flushInt time.Duration, maxSize int) *Batcher {
	c := resty.New().
		SetHeader("Content-Type", "application/json")
	return &Batcher{
		in:       make(chan models.Metrics, 1024),
		flushInt: flushInt,
		maxSize:  maxSize,
		client:   c,
		endpoint: endpoint,
	}
}

func (b *Batcher) Add(m models.Metrics) {
	// не блокируем продюсеров — канал с буфером; при переполнении лучше дропнуть/логировать
	select {
	case b.in <- m:
	default:
		logger.Log.Warn("batch channel full, metric dropped", zap.String("id", m.ID))
	}
}

func (b *Batcher) Run() {
	t := time.NewTicker(b.flushInt)
	defer t.Stop()

	buf := make([]models.Metrics, 0, b.maxSize)

	flush := func() {
		if len(buf) == 0 {
			return
		}
		payload, err := json.Marshal(buf)
		if err != nil {
			logger.Log.Error("marshal batch", zap.Error(err))
			buf = buf[:0]
			return
		}
		var gz bytes.Buffer
		zw := gzip.NewWriter(&gz)
		if _, err := zw.Write(payload); err != nil {
			logger.Log.Error("gzip write", zap.Error(err))
			buf = buf[:0]
			_ = zw.Close()
			return
		}
		_ = zw.Close()

		resp, err := b.client.R().
			SetHeader("Content-Encoding", "gzip").
			SetBody(gz.Bytes()).
			Post(b.endpoint)
		if err != nil {
			logger.Log.Error("batch post error", zap.Error(err))
			return
		}
		if resp.StatusCode() != http.StatusOK {
			logger.Log.Warn("batch non-200", zap.Int("status", resp.StatusCode()))
			return
		}
		buf = buf[:0] // очистили только при успехе
	}

	for {
		select {
		case m, ok := <-b.in:
			if !ok {
				flush()
				return
			}
			buf = append(buf, m)
			if b.maxSize > 0 && len(buf) >= b.maxSize {
				flush()
			}
		case <-t.C:
			flush()
		}
	}
}
