package handler

import (
	"net/http"
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
			)
		})
	}
}
