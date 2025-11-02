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

func NewStorageRepo(data map[string]string) *MapRepo {
	return &MapRepo{data: data}
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

func (r *MapRepo) StoreData() map[string]string {
	return r.data
}

func (r *MapRepo) LoadData() *map[string]string {
	return &r.data
}
