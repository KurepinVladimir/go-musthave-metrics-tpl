package cryptohelpers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign возвращает HMAC-SHA256 (в hex) от данных data с ключом key.
func Sign(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	_, _ = h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Compare возвращает true, если указанная hex‑подпись signedHex соответствует
// HMAC-SHA256 от данных data с ключом key.
func Compare(data []byte, key string, signedHex string) bool {
	expected := Sign(data, key)
	return hmac.Equal([]byte(expected), []byte(signedHex))
}
