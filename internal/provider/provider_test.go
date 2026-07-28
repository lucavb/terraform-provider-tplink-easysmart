package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPClientRetainsSessionCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "active"})
		case "/protected":
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value != "active" {
				http.Error(w, "missing session cookie", http.StatusUnauthorized)
				return
			}
		}
	}))
	defer server.Close()

	client, err := newHTTPClient(time.Second)
	if err != nil {
		t.Fatalf("newHTTPClient() error = %v", err)
	}

	for _, endpoint := range []string{"/login", "/protected"} {
		response, err := client.Get(server.URL + endpoint)
		if err != nil {
			t.Fatalf("GET %s error = %v", endpoint, err)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatalf("GET %s close body error = %v", endpoint, err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", endpoint, response.StatusCode, http.StatusOK)
		}
	}
}
