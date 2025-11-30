package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type URL struct {
	UUID     string `json:"uuid"`
	ShortURL string `json:"short_url"`
	URL      string `json:"original_url"`
}

type Repository interface {
	PutLink(url, shortURL string) error
	GetLink(shortURL string) (string, bool)
	LoadData() error
	SaveData() error
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
	PeriodicSave(time.Duration) <-chan error
}

type MapRepo struct {
	mu       sync.RWMutex
	data     map[string]URL
	filename string
}

type DBRepo struct {
	db *pgx.Conn
}

type DuplicateError struct {
	ShortURL string
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("short URL already exists: %s", e.ShortURL)
}

func NewMapRepo(db *pgx.Conn, storagePath string) (*MapRepo, error) {
	repo := &MapRepo{data: make(map[string]URL), filename: storagePath}
	if repo.filename == "" {
		return repo, nil
	}
	err := repo.LoadData()
	return repo, err
}

func NewDBRepo(db *pgx.Conn) (*DBRepo, error) {
	repo := &DBRepo{}
	if db != nil {
		repo.db = db
	}
	err := repo.LoadData()
	return repo, err

}

func (r *MapRepo) PutLink(url, shortURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	uuid := uuid.New().String()
	u := URL{UUID: uuid, ShortURL: shortURL, URL: url}
	r.data[shortURL] = u
	return nil
}

func (r *DBRepo) PutLink(url, shortURL string) error {
	uuid := uuid.New().String()
	u := URL{UUID: uuid, ShortURL: shortURL, URL: url}

	_, err := r.db.Exec(context.Background(),
		`insert into shorts (uuid, short_url, original_url) values ($1, $2, $3)`,
		u.UUID, u.ShortURL, u.URL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return &DuplicateError{ShortURL: shortURL}
			}
		}
	}
	return nil
}

func (r *DBRepo) GetLink(shortURL string) (string, bool) {
	var url string
	err := r.db.QueryRow(context.Background(),
		`select original_url from shorts where short_url = $1`,
		shortURL).Scan(&url)
	if err != nil {
		return "", false
	}
	return url, true
}

func (r *MapRepo) GetLink(shortURL string) (string, bool) {
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

func (r *DBRepo) LoadData() error {
	return nil
}

func (r *DBRepo) SaveData() error {
	return nil
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

func (r *DBRepo) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

func (r *MapRepo) Ping(ctx context.Context) error {
	return nil
}

func (r *MapRepo) Close(ctx context.Context) error {
	return nil
}

func (r *DBRepo) Close(ctx context.Context) error {
	return r.db.Close(ctx)
}

func (r *DBRepo) PeriodicSave(interval time.Duration) <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}
