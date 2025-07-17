package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5/middleware"
)

// Log будет доступен всему коду как синглтон.
// Никакой код навыка, кроме функции Initialize, не должен модифицировать эту переменную.
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	Log = zl
	return nil
}

func RequestLogger(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Обёртка для записи размера и кода ответа
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		start := time.Now()
		next.ServeHTTP(ww, r)
		duration := time.Since(start)

		//logger.
		Log.Info("incoming request",
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.Int("status", ww.Status()),
			zap.Int("size", ww.BytesWritten()),
			zap.Duration("duration", duration),
		)

	})
}
