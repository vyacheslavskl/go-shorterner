package handler

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vyacheslavskl/go-shorterner/internal/config"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
)

type ShorterHandler struct {
	cfg *config.Config
	srv *service.ShortServerice
}

func NewShorterHandler(cfg *config.Config, srv *service.ShortServerice) *ShorterHandler {
	return &ShorterHandler{cfg: cfg, srv: srv}
}

func (h *ShorterHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", h.GetRootLink)
	r.Post("/", h.PostLink)
	r.Get("/{id}", h.GetLink)

	return r
}

func (h *ShorterHandler) GetRootLink(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func (h *ShorterHandler) GetLink(w http.ResponseWriter, r *http.Request) {
	fURL := r.URL.Path
	res, ok := h.srv.GetLink(fURL[1:])
	if ok && res != "" {
		w.Header().Add("Location", res)
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (h *ShorterHandler) PostLink(w http.ResponseWriter, r *http.Request) {
	url, err := io.ReadAll(r.Body)
	if err != nil || string(url) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL := h.srv.SaveURL(string(url))
	response := h.cfg.RedirectAddress.String() + "/" + shortURL

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(response))
}
