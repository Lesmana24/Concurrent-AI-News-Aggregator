package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSafeHTTPClient_BlocksDirectPrivateDial(t *testing.T) {
	// Ensure ALLOW_INTERNAL_URLS is false
	t.Setenv("ALLOW_INTERNAL_URLS", "false")

	client := NewSafeHTTPClient(3 * time.Second)

	// Direct loopback IP
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:54321", nil)
	_, err := client.Do(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")

	// Direct private RFC 1918 IP
	reqPrivate, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://10.0.0.1:80", nil)
	_, errPrivate := client.Do(reqPrivate)
	assert.Error(t, errPrivate)
	assert.Contains(t, errPrivate.Error(), "blocked")
}

func TestSafeHTTPClient_BlocksMaliciousRedirect(t *testing.T) {
	// Mock server that returns a 302 redirect to an AWS/cloud metadata IP
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer ts.Close()

	// While ALLOW_INTERNAL_URLS is false, following redirect to 169.254.x.x must be blocked
	t.Setenv("ALLOW_INTERNAL_URLS", "false")
	client := NewSafeHTTPClient(3 * time.Second)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	_, err := client.Do(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestSafeHTTPClient_AllowsInternalWhenConfigured(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("internal-ok"))
	}))
	defer ts.Close()

	t.Setenv("ALLOW_INTERNAL_URLS", "true")
	client := NewSafeHTTPClient(3 * time.Second)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestScrapeArticle_EnforcesBodyLimit(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	// Create a test server that serves 6 MB of repeated paragraph tags (> 5 MB limit)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		chunk := "<p>Paragraf berita aktual yang valid dan informatif untuk pengujian.</p>\n"
		w.Write([]byte("<html><head><title>Berita Ukuran Raksasa</title></head><body>"))
		for i := 0; i < 90000; i++ {
			w.Write([]byte(chunk))
		}
		w.Write([]byte("</body></html>"))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo).(*newsService)

	scraped, err := svc.scrapeArticle(context.Background(), ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, "Berita Ukuran Raksasa", scraped.Title)
	// Content must be safely truncated without OOM or panic
	assert.NotEmpty(t, scraped.Content)
	assert.LessOrEqual(t, len(scraped.Content), 5*1024*1024)
}
