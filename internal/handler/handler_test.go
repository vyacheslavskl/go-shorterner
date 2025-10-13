package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vyacheslavskl/go-shorterner/internal/config"
)

func TestHander(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		contentType  string
		url          string
		body         io.Reader
		expectedCode int
	}{
		{name: "wrong method PUT",
			method:       http.MethodPut,
			url:          "/",
			contentType:  "text/plain",
			body:         nil,
			expectedCode: http.StatusBadRequest},
		{name: "simple POST",
			method:       http.MethodPost,
			contentType:  "text/plain",
			url:          "/",
			body:         strings.NewReader("ya.ru"),
			expectedCode: http.StatusCreated},
		// hash for ya.ru is 06509a58
		{name: "simple GET",
			method:       http.MethodGet,
			contentType:  "text/plain",
			url:          "/06509a58",
			body:         nil,
			expectedCode: http.StatusTemporaryRedirect},
		{name: "simple GET empty Url",
			method:       http.MethodGet,
			contentType:  "text/plain",
			url:          "/",
			body:         nil,
			expectedCode: http.StatusBadRequest},
	}

	for _, tc := range tests {
		cfg := new(config.Config)
		t.Run(tc.name, func(t *testing.T) {

			handler := PostRootHandler(cfg)
			r := httptest.NewRequest(tc.method, tc.url, tc.body)
			r.Header.Add("Content-Type", tc.contentType)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			res := w.Result()
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			if res.StatusCode != tc.expectedCode {
				t.Errorf("Result = %v, want %v", res.StatusCode, tc.expectedCode)
			}

			if tc.method == http.MethodPost && resBody == nil {
				t.Errorf("Body is not empty = %v", string(resBody))
			}

			if tc.method == http.MethodPost && err != nil {
				t.Errorf("Err is not nil = %v", err)
			}

		})

	}

}
