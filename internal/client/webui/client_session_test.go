package webui

import (
	"net/http"
	"testing"
	"time"
)

func TestEnsureSessionHTTPClientAddsCookieJar(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		client := ensureSessionHTTPClient(nil, 5*time.Second)
		if client.Jar == nil {
			t.Fatal("expected cookie jar on new client")
		}
		if client.Timeout != 5*time.Second {
			t.Fatalf("timeout = %s, want 5s", client.Timeout)
		}
	})

	t.Run("jar-less client", func(t *testing.T) {
		provided := &http.Client{Timeout: 3 * time.Second}
		client := ensureSessionHTTPClient(provided, 0)
		if client.Jar == nil {
			t.Fatal("expected cookie jar on cloned client")
		}
		if client.Timeout != 3*time.Second {
			t.Fatalf("timeout = %s, want 3s", client.Timeout)
		}
		if client == provided {
			t.Fatal("expected cloned client, got same pointer")
		}
	})
}
