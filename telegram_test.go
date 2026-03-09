package stt_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	stt "github.com/anatolykoptev/go-stt"
)

// sttHandler returns an httptest handler that serves a fake STT transcription response.
func sttHandler(t *testing.T, text string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stt.Response{Text: text, Language: "ru", Duration: 1.0})
	}
}

// audioHandler returns an httptest handler that serves fake audio bytes.
func audioHandler(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "audio/ogg")
		_, _ = w.Write(data)
	}
}

func TestTranscribeURL(t *testing.T) {
	audioSrv := httptest.NewServer(audioHandler([]byte("fake ogg audio")))
	defer audioSrv.Close()

	sttSrv := httptest.NewServer(sttHandler(t, "привет мир"))
	defer sttSrv.Close()

	client := stt.New(sttSrv.URL)
	resp, err := client.TranscribeURL(context.Background(), audioSrv.URL+"/voice.ogg")
	if err != nil {
		t.Fatalf("TranscribeURL: %v", err)
	}
	if resp.Text != "привет мир" {
		t.Errorf("text = %q, want %q", resp.Text, "привет мир")
	}
	if resp.Language != "ru" {
		t.Errorf("language = %q, want ru", resp.Language)
	}
}

func TestTranscribeURLVerbose(t *testing.T) {
	audioSrv := httptest.NewServer(audioHandler([]byte("fake ogg audio")))
	defer audioSrv.Close()

	sttSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stt.VerboseResponse{
			Text:     "тест",
			Language: "ru",
			Duration: 2.0,
			Segments: []stt.Segment{{ID: 0, Start: 0, End: 2.0, Text: "тест"}},
		})
	}))
	defer sttSrv.Close()

	client := stt.New(sttSrv.URL)
	resp, err := client.TranscribeURLVerbose(context.Background(), audioSrv.URL+"/voice.ogg")
	if err != nil {
		t.Fatalf("TranscribeURLVerbose: %v", err)
	}
	if resp.Text != "тест" {
		t.Errorf("text = %q, want тест", resp.Text)
	}
	if len(resp.Segments) != 1 {
		t.Errorf("segments = %d, want 1", len(resp.Segments))
	}
}

func TestTranscribeURLDownloadError(t *testing.T) {
	audioSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer audioSrv.Close()

	client := stt.New("http://127.0.0.1:1")
	_, err := client.TranscribeURL(context.Background(), audioSrv.URL+"/missing.ogg")
	if err == nil {
		t.Fatal("expected error for 404 audio download")
	}
	if !strings.Contains(err.Error(), "download audio") {
		t.Errorf("error = %q, want to contain 'download audio'", err.Error())
	}
}

func TestTranscribeURLTempCleanup(t *testing.T) {
	var capturedTempPath string

	// Intercept the STT request to check that the temp file exists during transcription.
	sttSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil { //nolint:mnd
			t.Errorf("parse multipart: %v", err)
		}
		// Extract the filename to reconstruct the temp path — we verify cleanup after.
		_, header, _ := r.FormFile("file")
		if header != nil {
			capturedTempPath = header.Filename
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stt.Response{Text: "done", Language: "ru"})
	}))
	defer sttSrv.Close()

	audioSrv := httptest.NewServer(audioHandler([]byte("ogg data")))
	defer audioSrv.Close()

	client := stt.New(sttSrv.URL)
	_, err := client.TranscribeURL(context.Background(), audioSrv.URL+"/voice.ogg")
	if err != nil {
		t.Fatalf("TranscribeURL: %v", err)
	}

	// The filename in the multipart form is the base name (e.g. "stt-voice-12345.ogg").
	// We can't recover the full path, so instead verify that no stt-voice-*.ogg files
	// remain in the OS temp directory after the call.
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "stt-voice-") && strings.HasSuffix(e.Name(), ".ogg") {
			t.Errorf("temp file not cleaned up: %s", e.Name())
		}
	}

	// Verify we captured the expected filename pattern.
	if capturedTempPath != "" {
		if !strings.HasPrefix(capturedTempPath, "stt-voice-") {
			t.Errorf("unexpected temp filename: %q", capturedTempPath)
		}
	}
}
