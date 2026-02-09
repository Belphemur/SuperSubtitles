package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/config"
)

func TestClient_CheckForUpdates(t *testing.T) {
	// Sample JSON response for update check
	jsonResponse := `{"film":2,"sorozat":5}`

	// Create a test server that returns the sample JSON for update check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "recheck" {
			t.Errorf("Expected action 'recheck', got %s", r.URL.Query().Get("action"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonResponse))
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call CheckForUpdates
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "1760700519")

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that we got the expected result
	if result == nil {
		t.Fatal("Expected result, got nil")
		return
	}

	// Test the counts
	if result.FilmCount != 2 {
		t.Errorf("Expected FilmCount 2, got %d", result.FilmCount)
	}
	if result.SeriesCount != 5 {
		t.Errorf("Expected SeriesCount 5, got %d", result.SeriesCount)
	}
	if !result.HasUpdates {
		t.Error("Expected HasUpdates to be true")
	}
}

func TestClient_CheckForUpdates_WithPrefix(t *testing.T) {
	// Sample JSON response for update check
	jsonResponse := `{"film":0,"sorozat":1}`

	// Create a test server that returns the sample JSON for update check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonResponse))
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call CheckForUpdates with "a_" prefix (should be trimmed)
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "a_1760700519")

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that we got the expected result
	if result == nil {
		t.Fatal("Expected result, got nil")
		return
	}

	// Test the counts
	if result.FilmCount != 0 {
		t.Errorf("Expected FilmCount 0, got %d", result.FilmCount)
	}
	if result.SeriesCount != 1 {
		t.Errorf("Expected SeriesCount 1, got %d", result.SeriesCount)
	}
	if !result.HasUpdates {
		t.Error("Expected HasUpdates to be true")
	}
}

func TestClient_CheckForUpdates_NoUpdates(t *testing.T) {
	// Sample JSON response for no updates
	jsonResponse := `{"film":0,"sorozat":0}`

	// Create a test server that returns the sample JSON for update check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonResponse))
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call CheckForUpdates
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "1760700519")

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that we got the expected result
	if result == nil {
		t.Fatal("Expected result, got nil")
		return
	}

	// Test the counts
	if result.FilmCount != 0 {
		t.Errorf("Expected FilmCount 0, got %d", result.FilmCount)
	}
	if result.SeriesCount != 0 {
		t.Errorf("Expected SeriesCount 0, got %d", result.SeriesCount)
	}
	if result.HasUpdates {
		t.Error("Expected HasUpdates to be false")
	}
}

func TestClient_CheckForUpdates_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call CheckForUpdates
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "1760700519")

	// Test that the call fails with an error
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestClient_CheckForUpdates_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call CheckForUpdates
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "1760700519")

	// Test that the call fails with JSON decode error
	if err == nil {
		t.Fatal("Expected JSON decode error, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}
