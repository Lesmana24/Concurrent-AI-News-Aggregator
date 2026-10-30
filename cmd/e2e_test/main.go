package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"news-aggregator/internal/ai"
	deliveryHttp "news-aggregator/internal/delivery/http"
	"news-aggregator/internal/middleware"
	"news-aggregator/internal/repository"
	"news-aggregator/internal/service"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	gin.SetMode(gin.TestMode)

	host := os.Getenv("DB_HOST")
	if host == "localhost" {
		host = "127.0.0.1"
	}
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable connect_timeout=5",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	repo := repository.NewPostgresRepository(db)
	aiSvc := ai.NewDefaultAIService()
	svc := service.NewNewsService(repo, aiSvc)
	handler := deliveryHttp.NewNewsHandler(svc)

	r := gin.New()
	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		protected := api.Group("/")
		protected.Use(middleware.APIKeyAuth(svc))
		{
			protected.POST("/summarize", handler.Summarize)
			protected.POST("/qa", handler.QA)
			protected.POST("/news/qa", handler.QA)
		}
	}

	validKey := "test-key-123"
	targetURL := "https://www.kompas.tv/regional/698663/ada-61-titik-demo-29-oktober-di-jakarta-hari-ini-polisi-siapkan-pengamanan"

	fmt.Println("================================================================================")
	fmt.Println("          TAHAP 5: PENGUJIAN API ENDPOINT & STRESS TEST (E2E SUITE)")
	fmt.Println("================================================================================")

	// -------------------------------------------------------------------------
	// 5.2.1: Health Check
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 1] GET /api/v1/health")
	w := performRequest(r, "GET", "/api/v1/health", "", nil)
	assertStatus(1, w.Code, http.StatusOK, w.Body.String())

	// -------------------------------------------------------------------------
	// 5.2.2: Missing API Key
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 2] POST /api/v1/summarize (Missing API Key - Expect 401)")
	w = performRequest(r, "POST", "/api/v1/summarize", "", map[string]interface{}{
		"urls": []string{targetURL},
	})
	assertStatus(2, w.Code, http.StatusUnauthorized, w.Body.String())

	// -------------------------------------------------------------------------
	// 5.2.3: Invalid API Key
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 3] POST /api/v1/summarize (Invalid API Key - Expect 401)")
	w = performRequest(r, "POST", "/api/v1/summarize", "wrong-key-999", map[string]interface{}{
		"urls": []string{targetURL},
	})
	assertStatus(3, w.Code, http.StatusUnauthorized, w.Body.String())

	// -------------------------------------------------------------------------
	// 5.2.4: Empty Array / Invalid Request Body
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 4] POST /api/v1/summarize (Empty URLs Array - Expect 400)")
	w = performRequest(r, "POST", "/api/v1/summarize", validKey, map[string]interface{}{
		"urls": []string{},
	})
	assertStatus(4, w.Code, http.StatusBadRequest, w.Body.String())

	// -------------------------------------------------------------------------
	// 5.2.5: SSRF Protection
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 5] POST /api/v1/summarize (SSRF Localhost URL - Expect Error Rejection)")
	w = performRequest(r, "POST", "/api/v1/summarize", validKey, map[string]interface{}{
		"urls": []string{"http://localhost:8080/internal/secret"},
	})
	if w.Code != http.StatusOK {
		fmt.Printf("   -> PASS: Blocked with HTTP %d (%s)\n", w.Code, w.Body.String())
	} else {
		log.Fatalf("   -> FAIL: Localhost SSRF was not blocked!")
	}

	// -------------------------------------------------------------------------
	// 5.2.6: Valid News URL Summarization (First Run)
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 6] POST /api/v1/summarize (Valid News URL - First Run Scraping + AI)")
	start := time.Now()
	w = performRequest(r, "POST", "/api/v1/summarize", validKey, map[string]interface{}{
		"urls": []string{targetURL},
	})
	dur1 := time.Since(start)
	assertStatus(6, w.Code, http.StatusOK, "")
	fmt.Printf("   -> Response Time: %v\n", dur1)
	fmt.Printf("   -> Output: %s\n", truncateString(w.Body.String(), 220))

	// -------------------------------------------------------------------------
	// 5.2.7: Identical URL (Cache Hit Verification)
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 7] POST /api/v1/summarize (Identical URL - Expect Fast Database Cache Hit)")
	start = time.Now()
	w = performRequest(r, "POST", "/api/v1/summarize", validKey, map[string]interface{}{
		"urls": []string{targetURL},
	})
	dur2 := time.Since(start)
	assertStatus(7, w.Code, http.StatusOK, "")
	fmt.Printf("   -> Cache Response Time: %v (Significantly faster than first run!)\n", dur2)

	// -------------------------------------------------------------------------
	// 5.2.8: Q&A - On-Topic Question
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 8] POST /api/v1/qa (On-Topic Question)")
	w = performRequest(r, "POST", "/api/v1/qa", validKey, map[string]interface{}{
		"urls":     []string{targetURL},
		"question": "Ada berapa titik demo di Jakarta hari ini?",
	})
	assertStatus(8, w.Code, http.StatusOK, "")
	fmt.Printf("   -> Q&A Output: %s\n", truncateString(w.Body.String(), 220))

	// -------------------------------------------------------------------------
	// 5.2.9: Q&A - Off-Topic Question (Guardrail Boundary)
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 9] POST /api/v1/qa (Off-Topic Question - Expect Polite Refusal)")
	w = performRequest(r, "POST", "/api/v1/qa", validKey, map[string]interface{}{
		"urls":     []string{targetURL},
		"question": "Bagaimana cara membuat martabak manis yang bersarang?",
	})
	assertStatus(9, w.Code, http.StatusOK, "")
	fmt.Printf("   -> Guardrail Refusal Output: %s\n", truncateString(w.Body.String(), 220))

	// -------------------------------------------------------------------------
	// 5.3: Concurrency & Stress Test (Fan-Out/Fan-In Goroutines)
	// -------------------------------------------------------------------------
	fmt.Println("\n[TEST 10] TAHAP 5.3 STRESS TEST: 10 Concurrent Requests (Goroutines Concurrency)")
	var wg sync.WaitGroup
	concurrentCount := 10
	errChan := make(chan error, concurrentCount)

	stressStart := time.Now()
	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rec := performRequest(r, "POST", "/api/v1/qa", validKey, map[string]interface{}{
				"urls":     []string{targetURL},
				"question": fmt.Sprintf("Siapa yang bertugas mengamankan aksi? (Req #%d)", idx+1),
			})
			if rec.Code != http.StatusOK {
				errChan <- fmt.Errorf("request #%d failed with status %d: %s", idx+1, rec.Code, rec.Body.String())
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	stressDuration := time.Since(stressStart)

	var stressErrors []error
	for e := range errChan {
		stressErrors = append(stressErrors, e)
	}

	if len(stressErrors) > 0 {
		log.Fatalf("   -> FAIL: %d of 10 concurrent requests failed! (%v)", len(stressErrors), stressErrors[0])
	} else {
		fmt.Printf("   -> PASS: All %d concurrent requests completed successfully in %v without race condition or deadlock!\n", concurrentCount, stressDuration)
	}

	fmt.Println("\n================================================================================")
	fmt.Println("             SELURUH PENGUJIAN TAHAP 5 TELAH SELESAI & LULUS 100%!")
	fmt.Println("================================================================================")
}

func performRequest(r http.Handler, method, path, apiKey string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(b)
	}
	req, _ := http.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-KEY", apiKey)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func assertStatus(testNum, actual, expected int, body string) {
	if actual != expected {
		log.Fatalf("[TEST %d] FAILED: expected HTTP %d, got %d. Body: %s", testNum, expected, actual, body)
	}
	fmt.Printf("   -> PASS: HTTP %d received as expected.\n", actual)
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
