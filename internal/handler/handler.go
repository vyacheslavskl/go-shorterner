package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vyacheslavskl/go-shorterner/internal/config"
	models "github.com/vyacheslavskl/go-shorterner/internal/model"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
	"go.uber.org/zap"
)

type ShorterHandler struct {
	cfg *config.Config
	srv *service.ShortServerice
	log *zap.SugaredLogger
}

func NewShorterHandler(cfg *config.Config, srv *service.ShortServerice, log *zap.SugaredLogger) *ShorterHandler {
	return &ShorterHandler{cfg: cfg, srv: srv, log: log}
}

func (h *ShorterHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(GzipMiddleware)
	r.Use(WithLogging(h.log))

	r.Get("/", h.GetRootLink)
	r.Post("/", h.PostLink)
	r.Get("/{id}", h.GetLink)
	r.Post("/api/shorten", h.PostAPIShorten)
	r.Post("/api/shorten/batch", h.PostAPIShortenBatch)
	r.Get("/ping", h.Ping)

	r.NotFound(h.GetRootLink)

	return r
}

func (h *ShorterHandler) GetRootLink(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func (h *ShorterHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.srv.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
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

func (h *ShorterHandler) PostAPIShorten(w http.ResponseWriter, r *http.Request) {
	// curl.exe -i -X  POST http://localhost:8080/api/shorten -H "Content-Type: application/json" -d '{\"url\": \"https://practicum.yandex.ru\"}'
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req models.Request
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shortURL := h.srv.SaveURL(req.URL)
	response := h.cfg.RedirectAddress.String() + "/" + shortURL

	resp := models.Response{Result: response}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// curl.exe -i -X  POST http://localhost:8080/api/shorten/batch -H "Content-Type: application/json" -d '[{\"correlation_id\": \"5deb996e-a5e3-4c56-bfe4-f5d4be91c5e9\", \"original_url\": \"sql.db\"}]'
func (h *ShorterHandler) PostAPIShortenBatch(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req []models.BatchRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var resp []models.BatchResponse
	for _, req := range req {
		resp = append(resp, models.BatchResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      h.cfg.RedirectAddress.String() + "/" + h.srv.SaveURL(req.OriginalURL),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
