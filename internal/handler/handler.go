package handler

import (
	"io"
	"net/http"

	"github.com/vyacheslavskl/go-shorterner/internal/config"
	"github.com/vyacheslavskl/go-shorterner/internal/repository"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
)

func PostRootHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fURL := r.URL.Path
			res, ok := repository.Repo[fURL[1:]]
			if ok && res != "" {
				w.Header().Add("Location", res)
				w.WriteHeader(http.StatusTemporaryRedirect)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}

			return
		} else if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			return
		} else if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {

			url, _ := io.ReadAll(r.Body)
			response := cfg.RedirectAddress.String() + "/" + service.Short(string(url))

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(response))
		}
	}
}
