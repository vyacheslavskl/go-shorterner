package service

import (
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

func (s *ShortServerice) PutLink(url, shortUrl string) {
	s.repo.PutLink(url, shortUrl)
}

func (s *ShortServerice) GetLink(shortUrl string) (string, bool) {
	return s.repo.GetLink(shortUrl)
}

func generateShort(url string) string {
	hasher := md5.New()
	hasher.Write([]byte(url))
	result := hex.EncodeToString(hasher.Sum(nil))
	return result[:8] // Берем первые 8 символов
}

func (s *ShortServerice) SaveUrl(u string) string {
	resultHash := generateShort(u)
	s.repo.PutLink(u, resultHash)
	// _, ok := repository.Repo[resultHash]

	// if ok {
	// 	return resultHash
	// }
	// repository.Repo[resultHash] = u

	return resultHash
}
