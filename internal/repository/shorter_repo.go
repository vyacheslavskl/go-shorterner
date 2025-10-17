package repository

import "sync"

type Repository interface {
	PutLink(url, shortURL string)
	GetLink(shortURL string) (string, bool)
}

type MapRepo struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMapRepo() *MapRepo {
	return &MapRepo{data: make(map[string]string)}
}

func (r *MapRepo) PutLink(url, shortURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[shortURL] = url
}

func (r *MapRepo) GetLink(shortURL string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, err := r.data[shortURL]
	return val, err
}
