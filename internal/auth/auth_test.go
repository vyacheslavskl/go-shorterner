package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTService(t *testing.T) {
	secretKey := []byte("my-super-secret-key-that-is-long-enough!")
	service := NewJWTService(secretKey)

	require.NotNil(t, service)
	assert.Equal(t, secretKey, service.SecretKey)
}

func TestNewJWTService_NilKey(t *testing.T) {
	service := NewJWTService(nil)

	require.NotNil(t, service)
	assert.Nil(t, service.SecretKey)
}

func TestNewJWTService_EmptyKey(t *testing.T) {
	service := NewJWTService([]byte(""))

	require.NotNil(t, service)
	assert.Equal(t, []byte(""), service.SecretKey)
}
