package handler

import (
	"compress/gzip"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type (
	responseData struct {
		status int
		size   int
	}

	responseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *responseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.responseData.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func WithLogging(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lw := &responseWriter{ResponseWriter: w,
				responseData: &responseData{
					status: 0,
					size:   0,
				}}

			h.ServeHTTP(lw, r)

			duration := time.Since(start)

			logger.Infow("HTTP request",
				"method", r.Method,
				"uri", r.RequestURI,
				"status", lw.responseData.status,
				"duration", duration,
				"size", lw.responseData.size,
				"content_type", r.Header.Get("Content-Type"),
			)
		})
	}
}

// curl.exe -i -X  POST http://localhost:8080/api/shorten -H "Content-Type: application/json" -H "Content-Encoding: gzip" -H "Accept-Encoding: gzip" -d '{\"url\": \"https://practicum.yandex.ru\"}'
// curl.exe -i -X  POST http://localhost:8080 -H "Content-Type: text/plain" -H "Content-Encoding: gzip" -H "Accept-Encoding: gzip" -d "ggg.rrr"
func WithGzipCompression(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}

		switch r.Header.Get("Content-Type") {
		case "application/json", "text/html":
			gz := gzip.NewWriter(w)
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")

			gzw := gzipResponseWriter{
				ResponseWriter: w,
				Writer:         gz,
			}
			h.ServeHTTP(gzw, r)
		default:
			h.ServeHTTP(w, r)
			return
		}
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

func (gzw gzipResponseWriter) Write(data []byte) (int, error) {
	return gzw.Writer.Write(data)
}
