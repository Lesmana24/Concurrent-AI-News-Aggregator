package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterURLs_Deduplication(t *testing.T) {
	raw := []string{
		"https://example.com/berita-1",
		"https://example.com/berita-1/",
		"https://example.com/berita-2",
	}

	filtered, err := FilterURLs(raw, 5)
	assert.NoError(t, err)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "https://example.com/berita-1", filtered[0])
	assert.Equal(t, "https://example.com/berita-2", filtered[1])
}

func TestFilterURLs_RejectLocalhostAndLoopback(t *testing.T) {
	raw := []string{"http://localhost:8080/secret"}
	_, err := FilterURLs(raw, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "localhost")

	rawIP := []string{"http://127.0.0.1:8080/internal"}
	_, errIP := FilterURLs(rawIP, 5)
	assert.Error(t, errIP)
	assert.Contains(t, errIP.Error(), "rejected")
}

func TestFilterURLs_RejectNonArticleExtensions(t *testing.T) {
	badURLs := []string{
		"https://example.com/document.pdf",
		"https://example.com/archive.zip",
		"https://example.com/photo.jpg",
		"https://example.com/program.exe",
	}

	for _, u := range badURLs {
		_, err := FilterURLs([]string{u}, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-article file type")
	}
}

func TestFilterURLs_RejectInvalidScheme(t *testing.T) {
	_, err := FilterURLs([]string{"ftp://example.com/news"}, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only http and https are allowed")
}

func TestFilterURLs_MaxCountLimit(t *testing.T) {
	raw := []string{
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/3",
		"https://example.com/4",
		"https://example.com/5",
		"https://example.com/6",
	}

	filtered, err := FilterURLs(raw, 5)
	assert.NoError(t, err)
	assert.Len(t, filtered, 5)
}

