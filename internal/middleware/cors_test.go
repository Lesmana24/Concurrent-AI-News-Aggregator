package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_PreflightOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for OPTIONS, got: %d", w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Fatalf("expected Access-Control-Allow-Origin: http://localhost:5173, got: %s", origin)
	}

	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Fatalf("expected Access-Control-Allow-Methods header to be set")
	}
}

func TestCORS_RegularRequestHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "success"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
		t.Fatalf("expected Access-Control-Allow-Origin: http://localhost:3000, got: %s", origin)
	}
}

