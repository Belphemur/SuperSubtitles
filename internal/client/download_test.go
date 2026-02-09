package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

func TestClient_DownloadSubtitle(t *testing.T) {
	// Test download of a regular (non-ZIP) subtitle file
	subtitleContent := "1\n00:00:01,000 --> 00:00:02,000\nTest subtitle line\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	result, err := client.DownloadSubtitle(ctx, server.URL, models.DownloadRequest{
		SubtitleID: "1234567890",
		Episode:    0,
	})

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
