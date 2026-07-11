package naming

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func geminiServer(t *testing.T, inner string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-goog-api-key"); got != "test-token" {
			t.Errorf("missing api key header, got %q", got)
		}
		w.WriteHeader(status)
		if status == http.StatusOK {
			envelope := map[string]any{
				"candidates": []map[string]any{
					{"content": map[string]any{"parts": []map[string]string{{"text": inner}}}},
				},
			}
			json.NewEncoder(w).Encode(envelope)
		} else {
			w.Write([]byte(`{"error":{"message":"nope"}}`))
		}
	}))
}

func testClient(url string) *Client {
	c := NewClient("test-token")
	c.BaseURL = url
	return c
}

func TestShouldParseBatchMatchedByOriginal(t *testing.T) {
	inner := `[
		{"original":"b.epub","type":"book","title":"B","author":"A"},
		{"original":"a.mkv","type":"movie","title":"A Movie","year":2001}
	]`
	srv := geminiServer(t, inner, http.StatusOK)
	defer srv.Close()

	got, err := testClient(srv.URL).Normalise(context.Background(), []string{"a.mkv", "b.epub"})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Title != "A Movie" || got[1].Title != "B" {
		t.Fatalf("expected order-independent matching by original, got %+v", got)
	}
}

func TestShouldErrorOnMalformedJSON(t *testing.T) {
	srv := geminiServer(t, `not json at all`, http.StatusOK)
	defer srv.Close()

	if _, err := testClient(srv.URL).Normalise(context.Background(), []string{"a.mkv"}); err == nil {
		t.Fatal("expected error on malformed inner JSON")
	}
}

func TestShouldErrorOnNon200(t *testing.T) {
	srv := geminiServer(t, "", http.StatusTooManyRequests)
	defer srv.Close()

	if _, err := testClient(srv.URL).Normalise(context.Background(), []string{"a.mkv"}); err == nil {
		t.Fatal("expected error on 429")
	}
}

func TestShouldHonourContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := testClient(srv.URL).Normalise(ctx, []string{"a.mkv"})
	if err == nil || time.Since(start) > time.Second {
		t.Fatalf("expected fast context-deadline error, got err=%v after %v", err, time.Since(start))
	}
}
