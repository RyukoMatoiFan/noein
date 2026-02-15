package speech

import (
	"noein/app/models"
	"testing"
)

func TestParseWhisperTimestampedOutput(t *testing.T) {
	out := `
main: processing 'x.wav'
[00:00:00.000 --> 00:00:01.000]   Hello
[00:00:01.050 --> 00:00:02.000]   world
[00:00:05.000 --> 00:00:06.000]   again
`

	segments, err := parseWhisperTimestampedOutput(out)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segments))
	}
	if segments[0].Text != "Hello" {
		t.Fatalf("unexpected first text: %q", segments[0].Text)
	}
}

func TestMergeTranscriptSegments(t *testing.T) {
	segments := []models.TranscriptSegment{
		{StartSec: 0.0, EndSec: 1.0, Text: "Hello"},
		{StartSec: 1.1, EndSec: 2.0, Text: "world"},
		{StartSec: 5.0, EndSec: 6.0, Text: "again"},
	}

	merged := mergeTranscriptSegments(segments, 0.5, 0.0)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged segments, got %d", len(merged))
	}
	if merged[0].Text != "Hello world" {
		t.Fatalf("unexpected merged text: %q", merged[0].Text)
	}
}
