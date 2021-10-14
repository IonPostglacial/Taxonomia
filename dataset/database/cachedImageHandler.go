package database

import (
	"fmt"
	"net/http"
)

func CachedImageHandler(reg *DatasetRegistry) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		src, ok := r.URL.Query()["src"]
		if !ok || len(src) != 1 {
			http.Error(w, fmt.Sprintf("The 'src' parameter is mandatory."), http.StatusBadRequest)
			return
		}
		if content, ok := reg.GetCachedImage(src[0]); ok {
			w.Write(content)
		} else {
			http.Error(w, fmt.Sprintf("Image not found: %q.", src[0]), http.StatusNotFound)
		}
	}
}
