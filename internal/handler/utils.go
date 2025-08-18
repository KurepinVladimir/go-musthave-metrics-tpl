package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/cryptohelpers"
)

// WriteSignedJSONResponse — сериализует m, подписывает его, и отправляет как JSON-ответ
func WriteSignedJSONResponse(w http.ResponseWriter, m any, key string) error {
	// сериализуем ответ сервера
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(m); err != nil {
		return err
	}

	if key != "" {
		hash := cryptohelpers.Sign(buf.Bytes(), key)
		w.Header().Set("HashSHA256", hash)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(buf.Bytes())
	return err
}
