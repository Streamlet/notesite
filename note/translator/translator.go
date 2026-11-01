package translator

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Translator interface {
	Translate() ([]byte, error)
}

type cacheEntry struct {
	modTime time.Time
	size    int64
	content []byte
}

var translateCache = struct {
	sync.RWMutex
	entries map[string]cacheEntry
}{entries: make(map[string]cacheEntry)}

// TranslateCached returns the translated output for path, reusing the previous
// result while the file's modification time and size are unchanged. The returned
// slice is shared with the cache and must not be modified.
func TranslateCached(path string) ([]byte, error) {
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	translateCache.RLock()
	entry, ok := translateCache.entries[path]
	translateCache.RUnlock()
	if ok && entry.size == info.Size() && entry.modTime.Equal(info.ModTime()) {
		return entry.content, nil
	}

	content, err := New(path).Translate()
	if err != nil {
		return nil, err
	}

	translateCache.Lock()
	translateCache.entries[path] = cacheEntry{
		modTime: info.ModTime(),
		size:    info.Size(),
		content: content,
	}
	translateCache.Unlock()
	return content, nil
}

func New(path string) Translator {
	switch filepath.Ext(path) {
	case ".md":
		return newMarkdownTranslator(path)
	case ".txt":
		return newTextTranslator(path)
	default:
		return newDefaultTranslator(path)
	}
}
