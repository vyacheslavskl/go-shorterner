package repository

import "sync"

type Repository interface {
	PutLink(url, shortUrl string)
	GetLink(shortUrl string) (string, bool)
}

type MapRepo struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMapRepo() *MapRepo {
	return &MapRepo{data: make(map[string]string)}
}

func (r *MapRepo) PutLink(url, shortUrl string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[shortUrl] = url
}

func (r *MapRepo) GetLink(shortUrl string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, err := r.data[shortUrl]
	return val, err
}
