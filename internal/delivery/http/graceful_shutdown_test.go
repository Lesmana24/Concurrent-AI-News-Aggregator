package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGracefulShutdown_DrainsInFlightRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	requestStarted := make(chan struct{})
	requestFinished := make(chan struct{})

	// Simulated slow in-flight endpoint (e.g. LLM streaming or heavy aggregation)
	router.GET("/slow", func(c *gin.Context) {
		close(requestStarted)
		time.Sleep(150 * time.Millisecond)
		close(requestFinished)
		c.JSON(http.StatusOK, gin.H{"status": "completed"})
	})

	// Bind to an ephemeral port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	serverAddr := listener.Addr().String()

	srv := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverStopped := make(chan struct{})
	go func() {
		err := srv.Serve(listener)
		assert.True(t, errors.Is(err, http.ErrServerClosed))
		close(serverStopped)
	}()

	client := &http.Client{Timeout: 3 * time.Second}

	var respStatus int
	var clientErr error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := client.Get("http://" + serverAddr + "/slow")
		if err != nil {
			clientErr = err
			return
		}
		defer resp.Body.Close()
		respStatus = resp.StatusCode
	}()

	// Wait until the slow request has arrived and begun processing
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request to start")
	}

	// Trigger graceful shutdown while the request is still actively running
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(shutdownCtx)
	assert.NoError(t, shutdownErr)

	// Wait for client request to finish
	wg.Wait()
	assert.NoError(t, clientErr)
	assert.Equal(t, http.StatusOK, respStatus, "in-flight request must be drained and succeed with 200 OK")

	// Verify the request finished completely
	select {
	case <-requestFinished:
	default:
		t.Fatal("request was not completed cleanly")
	}

	// Verify server stopped
	select {
	case <-serverStopped:
	case <-time.After(1 * time.Second):
		t.Fatal("server failed to stop")
	}

	// Verify new incoming requests after shutdown are rejected
	_, rejectErr := client.Get("http://" + serverAddr + "/slow")
	assert.Error(t, rejectErr, "new requests after shutdown must be rejected")
}

func TestGracefulShutdown_TimeoutExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/hangs", func(c *gin.Context) {
		time.Sleep(1 * time.Second)
		c.JSON(http.StatusOK, gin.H{"status": "done"})
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	srv := &http.Server{
		Handler: router,
	}

	go func() {
		_ = srv.Serve(listener)
	}()

	client := &http.Client{Timeout: 2 * time.Second}

	go func() {
		_, _ = client.Get("http://" + listener.Addr().String() + "/hangs")
	}()

	time.Sleep(30 * time.Millisecond)

	// Very short shutdown timeout (10ms) that is exceeded by the 1s request
	shortShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	shutdownErr := srv.Shutdown(shortShutdownCtx)
	assert.Error(t, shutdownErr, "expected context deadline exceeded error")
	assert.True(t, errors.Is(shutdownErr, context.DeadlineExceeded))
}

