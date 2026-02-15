package speech

import (
	"noein/app/models"
	"strings"
	"testing"
)

func TestRenderTranscriptSRT(t *testing.T) {
	segments := []models.TranscriptSegment{
		{StartSec: 0.0, EndSec: 1.234, Text: "Hello"},
		{StartSec: 1.5, EndSec: 2.0, Text: "world"},
	}

	data, ext, err := RenderTranscript("srt", segments)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ext != "srt" {
		t.Fatalf("expected ext srt, got %q", ext)
	}
	s := string(data)
	if !strings.Contains(s, "00:00:00,000 --> 00:00:01,234") {
		t.Fatalf("unexpected SRT time format: %s", s)
	}
}

func TestRenderTranscriptVTT(t *testing.T) {
	segments := []models.TranscriptSegment{
		{StartSec: 0.0, EndSec: 1.234, Text: "Hello"},
	}

	data, ext, err := RenderTranscript("vtt", segments)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ext != "vtt" {
		t.Fatalf("expected ext vtt, got %q", ext)
	}
	s := string(data)
	if !strings.HasPrefix(s, "WEBVTT") {
		t.Fatalf("expected WEBVTT header, got: %s", s)
	}
	if !strings.Contains(s, "00:00:00.000 --> 00:00:01.234") {
		t.Fatalf("unexpected VTT time format: %s", s)
	}
}
