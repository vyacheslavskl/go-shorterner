package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	models "github.com/vyacheslavskl/go-shorterner/internal/model"
	"github.com/vyacheslavskl/go-shorterner/internal/repository"
)

type ShortServerice struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) *ShortServerice {
	return &ShortServerice{repo: repo}
}

func (s *ShortServerice) PutLink(ctx context.Context, url, shortURL, userID string) {
	s.repo.PutLink(ctx, url, shortURL, userID)
}

func (s *ShortServerice) GetLink(ctx context.Context, shortURL string) (string, bool) {
	return s.repo.GetLink(ctx, shortURL)
}

func (s *ShortServerice) GetUserUrls(ctx context.Context, userID string) ([]models.UserUrlsResponse, bool) {
	return s.repo.GetUserUrls(ctx, userID)
}

func generateShort(url string) string {
	hasher := md5.New()
	hasher.Write([]byte(url))
	result := hex.EncodeToString(hasher.Sum(nil))
	return result[:8] // Берем первые 8 символов
}

func (s *ShortServerice) SaveURL(ctx context.Context, u, userID string) (string, error) {
	resultHash := generateShort(u)
	err := s.repo.PutLink(ctx, u, resultHash, userID)

	return resultHash, err
}

func (s *ShortServerice) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
