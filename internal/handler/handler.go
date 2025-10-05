package handler

import (
	"io"
	"net/http"

	"github.com/vyacheslavskl/go-shorterner/internal/repository"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
)

const localHostURL = "http://localhost:8080/"

func PostRoot(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		fURL := r.URL.Path
		res, ok := repository.Repo[fURL[1:]]
		if ok {
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

		response := localHostURL + service.Short(string(url))

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(response))
	}
}
