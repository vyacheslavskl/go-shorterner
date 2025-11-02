package handler

import (
	"compress/gzip"
	"io"
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

	compressWriter struct {
		w  http.ResponseWriter
		zw *gzip.Writer
	}

	compressReader struct {
		r  io.ReadCloser
		zr *gzip.Reader
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
				"content_encoding", r.Header.Get("Content-Encoding"),
				"accept_encoding", r.Header.Get("Accept-Encoding"),
				"response_header", lw.ResponseWriter.Header(),
			)
		})
	}
}

// curl.exe -i -X  POST http://localhost:8080/api/shorten -H "Content-Type: application/json" -H "Content-Encoding: gzip" -H "Accept-Encoding: gzip" -d '{\"url\": \"https://practicum.yandex.ru\"}'
// curl.exe -i -X  POST http://localhost:8080 -H "Content-Type: text/plain" -H "Content-Encoding: gzip" -H "Accept-Encoding: gzip" -d "ggg.rrr"
// curl.exe -i -X  POST http://localhost:8080 -H "Content-Type: text/plain" -H "Content-Encoding: gzip" -H "Accept-Encoding: gzip" -d "ya.rrr"
func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	return c.zw.Close()
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func GzipMiddlware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		contentType := r.Header.Get("Content-Type")
		if supportsGzip && (contentType == "application/json" || contentType == "text/html")  {
			cw := newCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip && (contentType == "application/json" || contentType == "text/html") {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	})
}
