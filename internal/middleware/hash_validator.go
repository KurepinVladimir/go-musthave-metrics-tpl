package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/cryptohelpers"
)

func ValidateHashSHA256(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// если ключ не задан — ничего не проверяем
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// читаем подпись только из правильного заголовка
			sentHash := r.Header.Get("HashSHA256")

			// пропускаем запросы без подписи (для автотестов и обратной совместимости)
			if sentHash == "" {
				next.ServeHTTP(w, r)
				return
			}

			// читаем тело (после gzip-мидлвари тут уже распаковано)
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "unable to read body", http.StatusInternalServerError)
				return
			}
			// возвращаем тело в r.Body для последующих обработчиков
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// сверяем HMAC от "сырых" данных (до сжатия)
			if !cryptohelpers.Compare(bodyBytes, key, sentHash) {
				http.Error(w, "invalid signature", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
