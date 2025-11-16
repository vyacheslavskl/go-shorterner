package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type URL struct {
	UUID     string `json:"uuid"`
	ShortURL string `json:"short_url"`
	URL      string `json:"original_url"`
}

type Repository interface {
	PutLink(url, shortURL string)
	GetLink(shortURL string) (string, bool)
	LoadData() error
	SaveData() error
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}

type MapRepo struct {
	db       *pgx.Conn
	mu       sync.RWMutex
	data     map[string]URL
	filename string
}

func NewMapRepo(db *pgx.Conn, storagePath string) (*MapRepo, error) {
	repo := &MapRepo{data: make(map[string]URL), filename: storagePath}
	if db != nil {
		repo.db = db
	}

	if repo.filename == "" {
		return repo, nil
	}

	err := repo.LoadData()
	return repo, err
}

func (r *MapRepo) PutLink(url, shortURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	uuid := uuid.New().String()
	u := URL{UUID: uuid, ShortURL: shortURL, URL: url}
	r.data[shortURL] = u
}

func (r *MapRepo) GetLink(shortURL string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, err := r.data[shortURL]
	return val.URL, err
}

func (r *MapRepo) saveStorage() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	values := make([]URL, 0, len(r.data))
	for _, value := range r.data {
		values = append(values, value)
	}

	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(r.filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r *MapRepo) loadStorage() error {
	file, err := os.ReadFile(r.filename)
	if err != nil {
		if os.IsNotExist(err) {
			initialData := []byte("[]")
			err = os.WriteFile(r.filename, initialData, 0666)
			if err != nil {
				return err
			} else {
				return nil
			}
		} else {
			return err
		}

	}
	var values []URL
	err = json.Unmarshal(file, &values)
	if err != nil {
		return err
	}
	for _, val := range values {
		r.data[val.ShortURL] = val
	}
	return nil
}

func (r *MapRepo) SaveData() error {
	err := r.saveStorage()
	if err != nil {
		return err
	}
	return nil
}

func (r *MapRepo) LoadData() error {
	err := r.loadStorage()
	return err
}

func (r *MapRepo) PeriodicSave(interval time.Duration) <-chan error {
	ticker := time.NewTicker(interval)
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		for range ticker.C {
			if err := r.SaveData(); err != nil {
				errCh <- err
			}
		}
	}()
	return errCh
}

func (r *MapRepo) Ping(ctx context.Context) error {
	if r.db == nil {
		return errors.New("nil repo")
	}
	return r.db.Ping(ctx)
}

func (r *MapRepo) Close(ctx context.Context) error {
	return r.db.Close(ctx)
}
