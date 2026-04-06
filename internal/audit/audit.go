package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"

	models "github.com/vyacheslavskl/go-shorterner/internal/model"
)

const (
	AuditActionShorten = "shorten"
	AuditActionFollow  = "follow"
)

type Observer interface {
	OnEvent(ctx context.Context, e models.AuditEvent) error
}

type Publisher struct {
	observers []Observer
}

func NewPublisher() *Publisher {
	return &Publisher{}
}

func (p *Publisher) Subscribe(o Observer) {
	p.observers = append(p.observers, o)
}

func (p *Publisher) Notify(ctx context.Context, e models.AuditEvent) {
	for _, o := range p.observers {
		obs := o
		go func() {
			_ = obs.OnEvent(ctx, e)
		}()
	}
}

type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

func NewFileObserver(path string) (*FileObserver, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileObserver{file: f}, nil
}

func (f *FileObserver) OnEvent(ctx context.Context, e models.AuditEvent) error {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err = f.file.Write(append(data, '\n'))
	return err
}

type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{},
	}
}

func (h *HTTPObserver) OnEvent(ctx context.Context, e models.AuditEvent) error {
	body, _ := json.Marshal(e)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := h.client.Do(req)
	return err
}
