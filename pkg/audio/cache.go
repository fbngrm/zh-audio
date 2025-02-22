package audio

import (
	"os"
	"path"
	"strings"
)

type Cache struct {
	AudioCacheDir string
}

func (c *Cache) GetCachePath(query string) string {
	query = strings.ReplaceAll(query, " ", "")
	filename := ""
	for _, c := range query {
		filename += string(c)
		if len(filename) >= 150 { // possible collisions
			break
		}
	}
	return path.Join(c.AudioCacheDir, filename) + ".mp3"
}

func (c *Cache) IsInCache(src string) bool {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return false
	}
	return true
}
