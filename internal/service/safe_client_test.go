package service

import (
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"sync"
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

func TestSafeHTTPClient_BlocksRedirectToDangerousPorts(t *testing.T) {
	client := NewSafeHTTPClient(3 * time.Second)

	initialReq, _ := http.NewRequest("GET", "https://news.example.com/article", nil)
	via := []*http.Request{initialReq}

	dangerousTargets := []string{
		"http://target.com:5432/db",
		"http://target.com:6379/keys",
		"http://target.com:22/ssh",
		"http://target.com:25/mail",
		"http://target.com:2375/docker",
	}

	for _, target := range dangerousTargets {
		redirReq, _ := http.NewRequest("GET", target, nil)
		err := client.CheckRedirect(redirReq, via)
		assert.Error(t, err, "expected error for target %s", target)
		assert.Contains(t, err.Error(), "dangerous port")
	}
}

func TestSafeHTTPClient_BlocksRedirectLoop(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")
	client := NewSafeHTTPClient(3 * time.Second)

	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, ts.URL+"/loop", http.StatusFound)
	}))
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	_, err := client.Do(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stopped after 5 redirects")
}

func TestSafeHTTPClient_StripsSensitiveHeadersOnCrossDomainRedirect(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")
	client := NewSafeHTTPClient(3 * time.Second)

	var receivedHeaders http.Header
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	// Initial server with different port/host that redirects to targetServer
	initialServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetServer.URL, http.StatusFound)
	}))
	defer initialServer.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, initialServer.URL, nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("X-API-Key", "api-key-12345")
	req.Header.Set("Cookie", "session=abcd")
	req.Header.Set("User-Agent", "NewsAggregatorBot")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Sensitive headers must NOT be forwarded across different origins
	assert.Empty(t, receivedHeaders.Get("Authorization"))
	assert.Empty(t, receivedHeaders.Get("X-API-Key"))
	assert.Empty(t, receivedHeaders.Get("Cookie"))
	// Non-sensitive header preserved
	assert.Equal(t, "NewsAggregatorBot", receivedHeaders.Get("User-Agent"))
}

func TestSafeHTTPClient_BlocksRedirectToInternalResolvedDomain(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "false")
	client := NewSafeHTTPClient(3 * time.Second)

	initialReq, _ := http.NewRequest("GET", "https://news.example.com/article", nil)
	via := []*http.Request{initialReq}

	internalTargets := []string{
		"http://127.0.0.1:8080/internal",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/admin",
		"http://192.168.1.1/",
		"http://100.100.100.200/metadata",
		"http://metadata.google.internal/computeMetadata/v1/",
		"http://instance-data/metadata",
	}

	for _, target := range internalTargets {
		redirReq, _ := http.NewRequest("GET", target, nil)
		err := client.CheckRedirect(redirReq, via)
		assert.Error(t, err, "expected error for internal target %s", target)
		assert.Contains(t, err.Error(), "blocked")
	}
}

func TestSafeHTTPClient_BlocksUnsupportedSchemeRedirect(t *testing.T) {
	client := NewSafeHTTPClient(3 * time.Second)
	initialReq, _ := http.NewRequest("GET", "https://news.example.com/article", nil)
	via := []*http.Request{initialReq}

	schemes := []string{
		"ftp://example.com/file",
		"file:///etc/passwd",
		"gopher://example.com/",
	}

	for _, target := range schemes {
		redirReq, _ := http.NewRequest("GET", target, nil)
		err := client.CheckRedirect(redirReq, via)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported scheme")
	}
}

func TestScrapeArticle_RejectsContentLengthExceedingLimit(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	// Server reports Content-Length of 10 MB
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", "10485760") // 10 MB
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Hello</body></html>"))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo).(*newsService)

	_, err := svc.scrapeArticle(context.Background(), ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds 5 MB limit")
}

func TestScrapeArticle_RejectsNonHTMLContentType(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	// Server returns binary octet-stream
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("\x00\x01\x02\x03\x04\x05"))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo).(*newsService)

	_, err := svc.scrapeArticle(context.Background(), ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported article content-type")
}

func TestScrapeArticle_BlocksGzipBomb(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	// Server serves gzipped payload that expands beyond 5 MB
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		// Write repeated paragraph tags in compressed form
		chunk := []byte("<p>Konten artikel berita aktual untuk pengujian gzip bomb.</p>\n")
		gz.Write([]byte("<html><head><title>Gzip Bomb Test</title></head><body>"))
		for i := 0; i < 90000; i++ {
			gz.Write(chunk)
		}
		gz.Write([]byte("</body></html>"))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo).(*newsService)

	scraped, err := svc.scrapeArticle(context.Background(), ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, "Gzip Bomb Test", scraped.Title)
	// Extracted content must not exceed 5 MB
	assert.LessOrEqual(t, len(scraped.Content), 5*1024*1024)
}


func TestNewsService_HTTPClient_SharedInstanceNoReinstantiation(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo).(*newsService)

	initialClient := svc.httpClient
	assert.NotNil(t, initialClient)

	// Verify that scraping doesn't overwrite or replace svc.httpClient
	t.Setenv("ALLOW_INTERNAL_URLS", "true")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><head><title>Shared Client</title></head><body><p>Ini adalah konten artikel berita pengujian yang memiliki panjang lebih dari dua puluh karakter.</p></body></html>"))
	}))
	defer ts.Close()

	_, err := svc.scrapeArticle(context.Background(), ts.URL)
	assert.NoError(t, err)

	assert.Same(t, initialClient, svc.httpClient, "HTTP client instance must remain identical and shared")
}

func TestNewsService_WithHTTPClient(t *testing.T) {
	mockRepo := new(MockRepository)
	customClient := &http.Client{Timeout: 42 * time.Second}

	svc := NewNewsService(mockRepo)
	svc.WithHTTPClient(customClient)

	concrete := svc.(*newsService)
	assert.Same(t, customClient, concrete.httpClient)
}

func TestNewsService_HTTPClient_LazyInitThreadSafe(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><head><title>Lazy Init</title></head><body><p>Ini adalah konten artikel berita pengujian yang memiliki panjang lebih dari dua puluh karakter.</p></body></html>"))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	svc := &newsService{
		repo: mockRepo,
		// httpClient intentionally left nil to test lazy initialization
	}

	var wg sync.WaitGroup
	clients := make([]*http.Client, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			_, err := svc.scrapeArticle(context.Background(), ts.URL)
			assert.NoError(t, err)
			svc.mu.RLock()
			clients[idx] = svc.httpClient
			svc.mu.RUnlock()
		}()
	}

	wg.Wait()

	assert.NotNil(t, svc.httpClient)
	for i := 0; i < 10; i++ {
		assert.Same(t, svc.httpClient, clients[i], "All concurrent routines must share the single lazily-initialized client")
	}
}

func TestScrapeArticle_ConnectionReuse_KeepAlive(t *testing.T) {
	t.Setenv("ALLOW_INTERNAL_URLS", "true")

	// Track new TCP connection creation
	var connCount int
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><head><title>Test Article</title></head><body><p>Ini adalah konten artikel berita pengujian yang memiliki panjang lebih dari dua puluh karakter.</p></body></html>"))
	}))
	defer ts.Close()

	mockRepo := new(MockRepository)
	svc := NewNewsService(mockRepo).(*newsService)

	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			mu.Lock()
			if !connInfo.Reused {
				connCount++
			}
			mu.Unlock()
		},
	}

	// First request: dials new TCP connection
	ctx1 := httptrace.WithClientTrace(context.Background(), trace)
	scraped1, err := svc.scrapeArticle(ctx1, ts.URL+"/article1")
	assert.NoError(t, err)
	assert.Equal(t, "Test Article", scraped1.Title)

	// Second request to same host: should reuse the idle connection from pool
	ctx2 := httptrace.WithClientTrace(context.Background(), trace)
	scraped2, err := svc.scrapeArticle(ctx2, ts.URL+"/article2")
	assert.NoError(t, err)
	assert.Equal(t, "Test Article", scraped2.Title)

	// Third request
	ctx3 := httptrace.WithClientTrace(context.Background(), trace)
	scraped3, err := svc.scrapeArticle(ctx3, ts.URL+"/article3")
	assert.NoError(t, err)
	assert.Equal(t, "Test Article", scraped3.Title)

	mu.Lock()
	defer mu.Unlock()
	// Exactly 1 initial TCP connection dialed, the next 2 requests reused the persistent connection
	assert.Equal(t, 1, connCount, "TCP connection should be reused across sequential requests via connection pool")
}
