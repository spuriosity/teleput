package naming

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestShouldSendCompleteDatasetWithoutTools(t *testing.T) {
	var request geminiRequest
	var rawRequest map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&rawRequest); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(rawRequest)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &request); err != nil {
			t.Fatal(err)
		}
		envelope := map[string]any{"candidates": []map[string]any{{"content": map[string]any{"parts": []map[string]string{{"text": `[]`}}}}}}
		json.NewEncoder(w).Encode(envelope)
	}))
	defer srv.Close()

	_, err := testClient(srv.URL).Normalise(context.Background(), []string{"one.epub", "two.m4b"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rawRequest["tools"]; ok {
		t.Fatalf("expected request without tools, got %+v", rawRequest["tools"])
	}
	prompt := request.Contents[0].Parts[0].Text
	if !strings.Contains(prompt, `["one.epub","two.m4b"]`) {
		t.Fatalf("expected complete JSON dataset, got %s", prompt)
	}
}

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
	c.Backoff = []time.Duration{0, 0}
	return c
}

// flakyServer answers with the given statuses in turn, then succeeds.
func flakyServer(t *testing.T, inner string, statuses ...int) (*httptest.Server, *int) {
	t.Helper()
	calls := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= len(statuses) {
			w.WriteHeader(statuses[calls-1])
			w.Write([]byte(`{"error":{"message":"This model is currently experiencing high demand."}}`))
			return
		}
		envelope := map[string]any{"candidates": []map[string]any{{"content": map[string]any{"parts": []map[string]string{{"text": inner}}}}}}
		json.NewEncoder(w).Encode(envelope)
	})), &calls
}

func TestShouldRetryOverloadedThenSucceed(t *testing.T) {
	srv, calls := flakyServer(t, `[{"original":"a.mkv","type":"movie","title":"A","year":2001}]`, http.StatusServiceUnavailable, http.StatusServiceUnavailable)
	defer srv.Close()

	got, err := testClient(srv.URL).Normalise(context.Background(), []string{"a.mkv"})
	if err != nil {
		t.Fatal(err)
	}
	if *calls != 3 || got[0].Title != "A" {
		t.Fatalf("expected success on third attempt, got calls=%d parsed=%+v", *calls, got)
	}
}

func TestShouldGiveUpAfterAllAttempts(t *testing.T) {
	srv, calls := flakyServer(t, `[]`, 503, 503, 503, 503)
	defer srv.Close()

	_, err := testClient(srv.URL).Normalise(context.Background(), []string{"a.mkv"})
	if err == nil || *calls != 3 {
		t.Fatalf("expected failure after 3 attempts, got err=%v calls=%d", err, *calls)
	}
	if !strings.Contains(err.Error(), "high demand") || !strings.Contains(err.Error(), "3 attempts") {
		t.Fatalf("expected readable message with attempt count, got %q", err)
	}
}

func TestShouldNotRetryClientErrors(t *testing.T) {
	srv, calls := flakyServer(t, `[]`, http.StatusBadRequest)
	defer srv.Close()

	if _, err := testClient(srv.URL).Normalise(context.Background(), []string{"a.mkv"}); err == nil || *calls != 1 {
		t.Fatalf("expected a single failed attempt on 400, got err=%v calls=%d", err, *calls)
	}
}

func TestShouldRetryStalledAttempt(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
			return
		}
		envelope := map[string]any{"candidates": []map[string]any{{"content": map[string]any{"parts": []map[string]string{{"text": `[]`}}}}}}
		json.NewEncoder(w).Encode(envelope)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	c.AttemptTimeout = 100 * time.Millisecond
	if _, err := c.Normalise(context.Background(), []string{"a.mkv"}); err != nil || calls != 2 {
		t.Fatalf("expected stalled first attempt to be retried, got err=%v calls=%d", err, calls)
	}
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

func TestShouldParseBookSeriesMetadata(t *testing.T) {
	inner := `[{"original":"book.epub","type":"book","title":"The Dungeon Anarchist's Cookbook","author":"Matt Dinniman","series":"Dungeon Crawler Carl","series_index":3}]`
	srv := geminiServer(t, inner, http.StatusOK)
	defer srv.Close()

	got, err := testClient(srv.URL).Normalise(context.Background(), []string{"book.epub"})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Series != "Dungeon Crawler Carl" || got[0].SeriesIndex != 3 {
		t.Fatalf("expected series metadata, got %+v", got[0])
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
