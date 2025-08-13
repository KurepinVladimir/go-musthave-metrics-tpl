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

	"context"
	"errors"
	"fmt"
	"net"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/retry"
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

		// Отправляем пакет метрик на сервер
		if err := b.postJSONWithRetry(context.Background(), b.endpoint, gz.Bytes()); err != nil {
			logger.Log.Error("batch post error", zap.Error(err))
			return
		}
		buf = buf[:0]
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

/////////////////////////////////

func (b *Batcher) postJSONWithRetry(ctx context.Context, url string, body []byte) error {
	return retry.DoIf(ctx, httpDelays, func(ctx context.Context) error {
		resp, err := b.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(body).
			Post(url)
		if err != nil {
			return err
		}
		if resp.StatusCode() == http.StatusBadGateway ||
			resp.StatusCode() == http.StatusServiceUnavailable ||
			resp.StatusCode() == http.StatusGatewayTimeout {
			return fmt.Errorf("temporary server error %d", resp.StatusCode())
		}
		if resp.StatusCode() >= 400 && resp.StatusCode() < 500 {
			return fmt.Errorf("client error %d: %s", resp.StatusCode(), resp.String())
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("server error %d: %s", resp.StatusCode(), resp.String())
		}
		return nil
	}, func(err error) bool {
		if err == nil {
			return false
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return true
		}
		// обрыв соединения и прочие транспортные ошибки — ретраим
		return true
	})
}
