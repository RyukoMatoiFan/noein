package llm

import "testing"

func TestParseCaptionTags_JSON(t *testing.T) {
	got, ok := ParseCaptionTags(`{"label":"person talking","tags":["speech","indoor","closeup"]}`)
	if !ok {
		t.Fatalf("expected ok")
	}
	if got.Label != "person talking" {
		t.Fatalf("unexpected label: %q", got.Label)
	}
	if len(got.Tags) != 3 {
		t.Fatalf("unexpected tags: %#v", got.Tags)
	}
}

func TestParseCaptionTags_WrappedText(t *testing.T) {
	got, ok := ParseCaptionTags("here you go:\n```json\n{\"label\":\"a\",\"tags\":[\"x\"]}\n```")
	if !ok {
		t.Fatalf("expected ok")
	}
	if got.Label != "a" || len(got.Tags) != 1 || got.Tags[0] != "x" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestParseCaptionTags_Fallback(t *testing.T) {
	got, ok := ParseCaptionTags("just a label")
	if !ok {
		t.Fatalf("expected ok")
	}
	if got.Label != "just a label" {
		t.Fatalf("unexpected label: %q", got.Label)
	}
}

