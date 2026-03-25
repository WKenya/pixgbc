package review

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/WKenya/pixgbc/internal/core"
)

func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func HashConfig(cfg core.Config) (string, error) {
	normalized, err := core.NormalizeConfig(cfg)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	return HashBytes(data), nil
}
