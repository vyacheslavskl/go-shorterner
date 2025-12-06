package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"github.com/vyacheslavskl/go-shorterner/internal/repository"
)

type ShortServerice struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) *ShortServerice {
	return &ShortServerice{repo: repo}
}

func (s *ShortServerice) PutLink(url, shortURL string) {
	s.repo.PutLink(url, shortURL)
}

func (s *ShortServerice) GetLink(shortURL string) (string, bool) {
	return s.repo.GetLink(shortURL)
}

func generateShort(url string) string {
	hasher := md5.New()
	hasher.Write([]byte(url))
	result := hex.EncodeToString(hasher.Sum(nil))
	return result[:8] // Берем первые 8 символов
}

func (s *ShortServerice) SaveURL(u string) (string, error) {
	resultHash := generateShort(u)
	err := s.repo.PutLink(u, resultHash)

	return resultHash, err
}

func (s *ShortServerice) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
