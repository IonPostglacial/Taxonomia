package dataset

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type fileCache struct {
	mu    sync.Mutex
	cache map[string][]byte
}

func (fc *fileCache) Get(path string) ([]byte, bool) {
	fc.mu.Lock()
	content, ok := fc.cache[path]
	fc.mu.Unlock()
	return content, ok
}

func (fc *fileCache) Set(path string, content []byte) {
	fc.mu.Lock()
	fc.cache[path] = content
	fc.mu.Unlock()
}

var staticFileCache *fileCache = &fileCache{cache: map[string][]byte{}}

func StaticHandler(w http.ResponseWriter, r *http.Request) {
	content, ok := staticFileCache.Get(r.URL.Path)
	if !ok {
		http.Error(w, fmt.Sprintf("File not found: %q.", r.URL.Path), http.StatusNotFound)
		return
	}
	if strings.HasSuffix(r.URL.Path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	w.Write(content)
}
