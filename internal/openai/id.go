package openai

import (
	"crypto/rand"
	"encoding/hex"
)

func NewResponseID() string {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		return "chatcmpl-fallback"
	}
	return "chatcmpl-" + hex.EncodeToString(buf)
}
