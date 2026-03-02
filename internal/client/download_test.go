package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
)

func TestClient_DownloadSubtitle(t *testing.T) {
	t.Parallel()
	// Test download of a regular (non-ZIP) subtitle file
	subtitleContent := "1\n00:00:01,000 --> 00:00:02,000\nTest subtitle line\n"
	expectedSubtitleID := "1234567890"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.php" {
			t.Errorf("Expected path '/index.php', got '%s'", r.URL.Path)
		}
		if action := r.URL.Query().Get("action"); action != "letolt" {
			t.Errorf("Expected action 'letolt', got '%s'", action)
		}
		if subtitleID := r.URL.Query().Get("felirat"); subtitleID != expectedSubtitleID {
			t.Errorf("Expected subtitle ID '%s', got '%s'", expectedSubtitleID, subtitleID)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(subtitleContent))
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	client := NewClient(testConfig)
	ctx := context.Background()

	result, err := client.DownloadSubtitle(ctx, expectedSubtitleID, nil)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
		return
	}
	if string(result.Content) != subtitleContent {
		t.Errorf("Expected content %q, got %q", subtitleContent, string(result.Content))
	}
	if result.ContentType == "" {
		t.Error("Expected ContentType to be set")
	}
	if result.Filename == "" {
		t.Error("Expected Filename to be set")
	}
}
