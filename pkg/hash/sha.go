package hash

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

func Sha1(s string) string {
	h := sha1.New()
	h.Write([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(h.Sum(nil))
}
