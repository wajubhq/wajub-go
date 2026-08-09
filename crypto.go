package wajub

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func hmacSha256(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func timingSafeEqual(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}
