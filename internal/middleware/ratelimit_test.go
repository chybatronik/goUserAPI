package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSecurityRateLimit(t *testing.T) {
	// Create rate limiter for testing - 10 requests per second with burst of 3
	limiter := SecurityRateLimit(10.0, 3)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Send rapid requests to test burst capacity
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		handler.ServeHTTP(w, req)

		if i < 3 {
			// First 3 should pass (burst capacity)
			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
			}
		} else {
			// Requests 4-5 should be rate limited
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: Expected status 429, got %d", i+1, w.Code)
			}
		}
	}
}

func TestRateLimitPerIP(t *testing.T) {
	limiter := SecurityRateLimit(100.0, 10)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with different IPs
	testCases := []struct {
		ip       string
		expected int
	}{
		{"192.168.1.1:12345", http.StatusOK},
		{"10.0.0.1:54321", http.StatusOK},
		{"172.16.0.1:9999", http.StatusOK},
	}

	for _, tc := range testCases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = tc.ip

		handler.ServeHTTP(w, req)

		if w.Code != tc.expected {
			t.Errorf("IP %s: Expected status %d, got %d", tc.ip, tc.expected, w.Code)
		}
	}
}

func TestRateLimitConcurrentRequests(t *testing.T) {
	limiter := SecurityRateLimit(50.0, 20) // 50 requests/sec, burst 20
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	rateLimitedCount := 0

	// Send 30 concurrent requests from the same IP
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.100:12345"

			handler.ServeHTTP(w, req)

			mu.Lock()
			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Some requests should succeed, some should be rate limited
	if successCount == 0 {
		t.Error("No requests succeeded, expected some to pass")
	}
	if rateLimitedCount == 0 {
		t.Error("No requests were rate limited, expected some to be limited")
	}
	t.Logf("Success: %d, Rate limited: %d", successCount, rateLimitedCount)
}

func TestRateLimitCleanup(t *testing.T) {
	limiter := SecurityRateLimit(1.0, 1) // 1 request/sec, burst 1
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send first request - should pass
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.200:12345"
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request should pass, got status %d", w1.Code)
	}

	// Send second request immediately - should be rate limited
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.200:12345"
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got status %d", w2.Code)
	}

	// Wait for rate limit to reset and send third request - should pass
	time.Sleep(1100 * time.Millisecond) // Wait longer than 1 second

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "192.168.1.200:12345"
	handler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Third request after wait should pass, got status %d", w3.Code)
	}
}

func TestRateLimitErrorResponse(t *testing.T) {
	limiter := SecurityRateLimit(1.0, 1)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send two rapid requests to trigger rate limiting
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.50:12345"
	handler.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.50:12345"
	handler.ServeHTTP(w2, req2)

	// Check rate limit response
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w2.Code)
	}

	// Check response body contains rate limit error
	body := w2.Body.String()
	if !strings.Contains(body, "Too many requests") {
		t.Errorf("Expected 'Too many requests' in response body, got: %s", body)
	}

	// Check response headers
	if contentType := w2.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got: %s", contentType)
	}
}

func TestRateLimitZeroRate(t *testing.T) {
	limiter := SecurityRateLimit(0, 0) // Zero rate should allow all requests
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send multiple rapid requests - all should pass
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.254:12345"
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200 with zero rate limit, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitHeaderParsing(t *testing.T) {
	limiter := SecurityRateLimit(10.0, 5)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	testCases := []struct {
		remoteAddr string
		expectedIP string
	}{
		{"192.168.1.1:12345", "192.168.1.1"},
		{"10.0.0.1:8080", "10.0.0.1"},
		{"172.16.0.1:9000", "172.16.0.1"},
		{"localhost:8080", "localhost"},
	}

	for _, tc := range testCases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = tc.remoteAddr

		handler.ServeHTTP(w, req)

		// Should pass since each is a different IP
		if w.Code != http.StatusOK {
			t.Errorf("IP %s: Expected status 200, got %d", tc.expectedIP, w.Code)
		}
	}
}
