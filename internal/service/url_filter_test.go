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

func TestFilterURLs_RejectCloudMetadataAndRestrictedIPs(t *testing.T) {
	restrictedURLs := []string{
		"http://metadata.google.internal/computeMetadata/v1/",
		"http://instance-data/latest/meta-data/",
		"http://169.254.169.254/latest/meta-data/",
		"http://100.100.100.200/latest/meta-data/", // Alibaba cloud metadata
		"http://100.64.0.1/internal-status",        // CGNAT
		"http://192.0.2.1/docs",                    // TEST-NET-1
		"http://[::ffff:127.0.0.1]:8080/admin",     // IPv4-mapped loopback
		"http://[::ffff:169.254.169.254]/meta",     // IPv4-mapped link-local
		"http://[::ffff:10.0.0.1]/secret",          // IPv4-mapped RFC 1918
	}

	for _, u := range restrictedURLs {
		_, err := FilterURLs([]string{u}, 5)
		assert.Error(t, err, "URL '%s' must be rejected by SSRF filter", u)
	}
}

func TestFilterURLs_RejectUnresolvableDomain(t *testing.T) {
	badDomain := "http://this-domain-definitely-does-not-exist-123456789.org/news"
	_, err := FilterURLs([]string{badDomain}, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not be resolved")
}

