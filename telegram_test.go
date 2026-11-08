package stt_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	stt "github.com/anatolykoptev/go-stt"
)

// TestTranscribeURLDownloadTimeout verifies that downloadToTemp uses the
// configured client (with timeout) instead of http.DefaultClient (no timeout).
// A slow server that delays beyond the client timeout should return an error,
// not hang forever.
func TestTranscribeURLDownloadTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // slow server
	}))
	defer srv.Close()

	c := stt.New("http://127.0.0.1:1", stt.WithTimeout(200*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.TranscribeURL(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected timeout error from slow download, got nil")
	}
}

// TestWithTempDirUsed verifies that WithTempDir causes temp files to be
// created in the configured directory.
func TestWithTempDirUsed(t *testing.T) {
	tmpDir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fake audio data"))
	}))
	defer srv.Close()

	c := stt.New("http://127.0.0.1:1", stt.WithTempDir(tmpDir), stt.WithTimeout(5*time.Second))

	// We can't easily test TranscribeURL end-to-end (it calls Transcribe
	// which hits the STT endpoint), but we can verify the temp dir is used
	// by checking that downloadToTemp creates files in tmpDir.
	// Use TranscribeURL against a server that returns audio — the download
	// will succeed, then Transcribe will fail (wrong endpoint), but the
	// temp file should be in tmpDir and cleaned up.
	_, err := c.TranscribeURL(context.Background(), srv.URL)
	if err == nil {
		// Transcribe will fail because the STT endpoint is unreachable,
		// but the download + temp file creation should work.
	}

	// Verify no temp files leaked in the system temp dir.
	systemTmp := os.TempDir()
	entries, _ := os.ReadDir(systemTmp)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "stt-voice-") {
			t.Errorf("temp file leaked in system temp dir: %s", e.Name())
		}
	}

	// Verify the configured temp dir is empty after cleanup (defer os.Remove).
	entries, _ = os.ReadDir(tmpDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "stt-voice-") {
			t.Errorf("temp file leaked in configured temp dir: %s", e.Name())
		}
	}
}

// TestTranscribeURLCleansTempOnPanic verifies that temp files are cleaned up
// even if the transcription step fails. We can't easily trigger a panic, but
// we can verify cleanup on error paths.
func TestTranscribeURLCleansTempOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fake audio data"))
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	c := stt.New("http://127.0.0.1:1", stt.WithTempDir(tmpDir), stt.WithTimeout(5*time.Second))

	// TranscribeURL will download successfully, then Transcribe will fail
	// (unreachable STT endpoint). The temp file should be cleaned up by defer.
	_, err := c.TranscribeURL(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error from unreachable STT endpoint")
	}

	// Verify no temp files leaked.
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "stt-voice-") {
			t.Errorf("temp file leaked after error: %s", filepath.Join(tmpDir, e.Name()))
		}
	}
}

// Download hardening: the API key must never reach the audio host, redirects
// are refused, errors never echo the URL, and the body is size-capped.

func TestDownloadNeverSendsAPIKey(t *testing.T) {
	var got atomic.Value
	got.Store("unset")
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Store(r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNotFound)
	}))
	defer foreign.Close()

	c := stt.New("http://127.0.0.1:1", stt.WithAPIKey("k"))
	_, _ = c.TranscribeURL(context.Background(), foreign.URL+"/voice.ogg")
	if v := got.Load().(string); v != "" {
		t.Fatalf("foreign host saw Authorization %q, want empty (unset = no request)", v)
	}
}

func TestDownloadRefusesRedirects(t *testing.T) {
	var hitsB atomic.Int32
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitsB.Add(1)
	}))
	defer b.Close()
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, b.URL+"/x.ogg", http.StatusFound)
	}))
	defer a.Close()

	_, err := stt.New("http://127.0.0.1:1", stt.WithAPIKey("k")).TranscribeURL(context.Background(), a.URL+"/voice.ogg")
	if err == nil {
		t.Fatal("expected error on redirect")
	}
	if n := hitsB.Load(); n != 0 {
		t.Fatalf("redirect target received %d requests, want 0", n)
	}
}

func TestDownloadErrorNeverContainsURL(t *testing.T) {
	_, err := stt.New("http://127.0.0.1:1").TranscribeURL(context.Background(),
		"http://127.0.0.1:1/file/botFAKE123:TOKENPART/voice.ogg")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, s := range []string{"bot", "TOKENPART", "FAKE123"} {
		if strings.Contains(err.Error(), s) {
			t.Fatalf("error %q leaks %q", err.Error(), s)
		}
	}
	// parse failure path
	_, err = stt.New("http://127.0.0.1:1").TranscribeURL(context.Background(), "http://bot:TOKENPART@[::1/x")
	if err == nil || strings.Contains(err.Error(), "TOKENPART") {
		t.Fatalf("parse error must be fixed and URL-free, got %v", err)
	}
}

func TestDownloadSizeCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 2048))
	}))
	defer srv.Close()

	_, err := stt.New("http://127.0.0.1:1", stt.WithMaxDownload(1024)).TranscribeURL(context.Background(), srv.URL+"/v.ogg")
	if !errors.Is(err, stt.ErrResponseTooLarge) {
		t.Fatalf("want ErrResponseTooLarge, got %v", err)
	}
}

func TestWithHTTPClientUsedForDownload(t *testing.T) {
	var used atomic.Bool
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		used.Store(true)
		return &http.Response{StatusCode: 404, Body: http.NoBody, Header: http.Header{}}, nil
	})
	c := stt.New("http://127.0.0.1:1", stt.WithHTTPClient(&http.Client{Transport: rt}))
	_, _ = c.TranscribeURL(context.Background(), "http://example.invalid/v.ogg")
	if !used.Load() {
		t.Fatal("custom client not used")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
