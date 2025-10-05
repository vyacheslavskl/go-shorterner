package service

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/vyacheslavskl/go-shorterner/internal/repository"
)


func generateShort(url string) string {
	hasher := md5.New()
	hasher.Write([]byte(url))
	result := hex.EncodeToString(hasher.Sum(nil))
	return result[:8] // Берем первые 8 символов
}

func Short(u string) string {
	result_hash := generateShort(u)
	_, ok := repository.Repo[result_hash]

	if ok {
		return result_hash
	}
	repository.Repo[result_hash] = u

	return result_hash
}
