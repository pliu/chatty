package middleware

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pliu/chatty/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	// Mock next handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey)
		if userID == nil {
			t.Error("Expected userID in context")
		}
		if userID.(int) != 123 {
			t.Errorf("Expected userID 123, got %v", userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		cookieValue    string
		expectedStatus int
	}{
		{
			name:           "Valid Cookie",
			cookieValue:    auth.SignCookie("123"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid Signature",
			cookieValue:    "123|invalid_signature",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Value",
			cookieValue:    "not_an_int|signature", // Signature won't match anyway
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.AddCookie(&http.Cookie{Name: "user_id", Value: tt.cookieValue})
			rr := httptest.NewRecorder()

			AuthMiddleware(nextHandler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedStatus)
			}
		})
	}

	t.Run("Missing Cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		AuthMiddleware(nextHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v",
				rr.Code, http.StatusUnauthorized)
		}
	})
}

func TestLoggingMiddleware(t *testing.T) {
	// Mock next handler that returns 404
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	LoggingMiddleware(nextHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rr.Code, http.StatusNotFound)
	}
}

// MockHijacker implements http.Hijacker for testing
type MockHijacker struct {
	httptest.ResponseRecorder
}

func (m *MockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func TestLoggingMiddleware_Hijack(t *testing.T) {
	// Mock next handler that tries to hijack
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("ResponseWriter does not implement http.Hijacker")
			return
		}
		_, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("Hijack failed: %v", err)
		}
	})

	req := httptest.NewRequest("GET", "/", nil)
	// httptest.ResponseRecorder doesn't implement Hijacker, so we need a custom one
	// But since we are testing the middleware wrapper, we need to pass something that DOES implement it
	// to the middleware.

	// The middleware wraps the writer passed to ServeHTTP.
	// So we need to pass a MockHijacker to the middleware.

	mockWriter := &MockHijacker{ResponseRecorder: *httptest.NewRecorder()}

	LoggingMiddleware(nextHandler).ServeHTTP(mockWriter, req)
}
