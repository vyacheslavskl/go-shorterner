package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vyacheslavskl/go-shorterner/internal/auth"
	"github.com/vyacheslavskl/go-shorterner/internal/config"
	models "github.com/vyacheslavskl/go-shorterner/internal/model"
	"github.com/vyacheslavskl/go-shorterner/internal/repository"
	"github.com/vyacheslavskl/go-shorterner/internal/service"
	"go.uber.org/zap"
)

type ShorterHandler struct {
	cfg *config.Config
	srv *service.ShortServerice
	log *zap.SugaredLogger
	jwt *auth.JWTService
}

func NewShorterHandler(cfg *config.Config, srv *service.ShortServerice, log *zap.SugaredLogger, jwt *auth.JWTService) *ShorterHandler {
	return &ShorterHandler{cfg: cfg, srv: srv, log: log, jwt: jwt}
}

func (h *ShorterHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(GzipMiddleware)
	r.Use(WithLogging(h.log))
	r.Use(AuthMiddleware(h.jwt, h.log))

	r.Get("/", h.GetRootLink)
	r.Post("/", h.PostLink)
	r.Get("/{id}", h.GetLink)
	r.Post("/api/shorten", h.PostAPIShorten)
	r.Post("/api/shorten/batch", h.PostAPIShortenBatch)
	r.Get("/ping", h.Ping)
	r.Get("/api/user/urls", h.GetAPIUserUrls)
	r.Delete("/api/user/urls", h.DeleteAPIUserUrls)

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
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	fURL := r.URL.Path
	res, ok, isDeleted := h.srv.GetLink(r.Context(), fURL[1:], userID)
	h.log.Infof("Result GetLink: %s, %t, %t", res, ok, isDeleted)
	if isDeleted {
		w.WriteHeader(http.StatusGone)
		return
	}
	if ok && res != "" {
		w.Header().Add("Location", res)
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (h *ShorterHandler) PostLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	url, err := io.ReadAll(r.Body)
	if err != nil || string(url) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := h.srv.SaveURL(context.Background(), string(url), userID)
	response := h.cfg.RedirectAddress.String() + "/" + shortURL

	w.Header().Set("Content-Type", "text/plain")
	if err != nil {
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	w.Write([]byte(response))
}

func (h *ShorterHandler) PostAPIShorten(w http.ResponseWriter, r *http.Request) {
	// curl.exe -i -X  POST http://localhost:8080/api/shorten -H "Content-Type: application/json" -d '{\"url\": \"https://practicum.yandex.ru\"}'
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.Request
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	shortURL, err := h.srv.SaveURL(r.Context(), req.URL, userID)
	response := h.cfg.RedirectAddress.String() + "/" + shortURL
	resp := models.Response{Result: response}
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		h.log.Errorf("error inserting into DB", err)
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	} else {
		w.WriteHeader(http.StatusCreated)
	}

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
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req []models.BatchRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.Infow("Request", "body", req)
	var resp []models.BatchResponse
	for _, req := range req {
		answer, err := h.srv.SaveURL(r.Context(), req.OriginalURL, userID)
		if err != nil {
			resp = append(resp, models.BatchResponse{
				CorrelationID: req.CorrelationID,
				ShortURL:      "",
			})
		}
		resp = append(resp, models.BatchResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      h.cfg.RedirectAddress.String() + "/" + answer,
		})
	}
	h.log.Infow("Respond", "body", resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *ShorterHandler) GetAPIUserUrls(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	urls, ok := h.srv.GetUserUrls(r.Context(), userID)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 || urls == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	for i := range urls {
		urls[i].ShortURL = h.cfg.RedirectAddress.String() + "/" + urls[i].ShortURL
	}
	enc := json.NewEncoder(w)
	err := enc.Encode(urls)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *ShorterHandler) DeleteAPIUserUrls(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var shortURLs []string
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&shortURLs); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := h.srv.DeleteUserUrls(r.Context(), userID, shortURLs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
