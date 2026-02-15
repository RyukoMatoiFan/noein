package speech

import (
	"bytes"
	"fmt"
	"log"
	"noein/app/models"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"
)

type WhisperRunner struct {
	WhisperPath string
	ModelPath   string
}

func (w *WhisperRunner) TranscribeSegments(inputVideoPath string, extractAudio func(inputPath, outputWavPath string) error) ([]models.TranscriptSegment, error) {
	if strings.TrimSpace(w.WhisperPath) == "" {
		return nil, fmt.Errorf("whisper executable path is required")
	}
	if strings.TrimSpace(w.ModelPath) == "" {
		return nil, fmt.Errorf("whisper model path is required")
	}

	tempDir, err := os.MkdirTemp("", "noein_whisper_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	wavPath := tempDir + string(os.PathSeparator) + "audio.wav"
	if err := extractAudio(inputVideoPath, wavPath); err != nil {
		return nil, err
	}

	rawOutput, err := w.runWhisperCli(wavPath)
	if err != nil {
		return nil, err
	}

	segments, err := parseWhisperTimestampedOutput(rawOutput)
	if err != nil {
		return nil, err
	}
	for i, s := range segments {
		log.Printf("[WHISPER DEBUG] parsed segment %d: start=%.3f end=%.3f text=%q", i, s.StartSec, s.EndSec, s.Text)
	}

	out := make([]models.TranscriptSegment, 0, len(segments))
	for _, s := range segments {
		out = append(out, models.TranscriptSegment{
			StartSec: s.StartSec,
			EndSec:   s.EndSec,
			Text:     s.Text,
		})
	}

	return out, nil
}

func (w *WhisperRunner) DetectSpeechFragments(inputVideoPath string, frameRate float64, mergeGapMs int, minFragmentMs int, extractAudio func(inputPath, outputWavPath string) error) ([]models.SpeechFragment, error) {
	if mergeGapMs <= 0 {
		mergeGapMs = 400
	}
	if minFragmentMs < 0 {
		minFragmentMs = 0
	}

	segments, err := w.TranscribeSegments(inputVideoPath, extractAudio)
	if err != nil {
		return nil, err
	}

	merged := mergeTranscriptSegments(segments, float64(mergeGapMs)/1000.0, float64(minFragmentMs)/1000.0)
	fragments := make([]models.SpeechFragment, len(merged))
	for i := range merged {
		fragments[i] = models.SpeechFragment{
			ID:       uuid.NewString(),
			StartSec: merged[i].StartSec,
			EndSec:   merged[i].EndSec,
			Text:     merged[i].Text,
		}
		fragments[i].InFrame = int64(fragments[i].StartSec * frameRate)
		fragments[i].OutFrame = int64(fragments[i].EndSec*frameRate) + 1
		if fragments[i].OutFrame <= fragments[i].InFrame {
			fragments[i].OutFrame = fragments[i].InFrame + 1
		}
	}

	return fragments, nil
}

func (w *WhisperRunner) runWhisperCli(wavPath string) (string, error) {
	cmd := exec.Command(
		w.WhisperPath,
		"-m", w.ModelPath,
		"-f", wavPath,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper failed: %w (stderr: %s)", err, errBuf.String())
	}

	out := outBuf.String()
	if out == "" && errBuf.Len() > 0 {
		out = errBuf.String()
	} else if errBuf.Len() > 0 {
		out = out + "\n" + errBuf.String()
	}
	log.Printf("[WHISPER DEBUG] stdout len=%d, stderr len=%d", outBuf.Len(), errBuf.Len())
	log.Printf("[WHISPER DEBUG] raw output first 2000 chars:\n%s", truncate(out, 2000))
	return out, nil
}

type whisperSegment struct {
	StartSec float64
	EndSec   float64
	Text     string
}

var whisperLineRE = regexp.MustCompile(`(?m)^\[(\d{2}):(\d{2}):(\d{2})\.(\d{3})\s+-->\s+(\d{2}):(\d{2}):(\d{2})\.(\d{3})\]\s*(.*)$`)

func parseWhisperTimestampedOutput(output string) ([]whisperSegment, error) {
	matches := whisperLineRE.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no timestamped segments found in whisper output")
	}

	segments := make([]whisperSegment, 0, len(matches))
	for _, m := range matches {
		startSec, err := parseHMSmsToSeconds(m[1], m[2], m[3], m[4])
		if err != nil {
			return nil, err
		}
		endSec, err := parseHMSmsToSeconds(m[5], m[6], m[7], m[8])
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(m[9])
		if endSec <= startSec {
			continue
		}
		if text == "" {
			continue
		}
		// Skip Whisper non-speech markers
		if isWhisperMarker(text) {
			continue
		}

		segments = append(segments, whisperSegment{
			StartSec: startSec,
			EndSec:   endSec,
			Text:     text,
		})
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("no usable speech segments found in whisper output")
	}

	return segments, nil
}

func parseHMSmsToSeconds(hh, mm, ss, ms string) (float64, error) {
	h, err := strconv.Atoi(hh)
	if err != nil {
		return 0, fmt.Errorf("invalid hours %q", hh)
	}
	m, err := strconv.Atoi(mm)
	if err != nil {
		return 0, fmt.Errorf("invalid minutes %q", mm)
	}
	s, err := strconv.Atoi(ss)
	if err != nil {
		return 0, fmt.Errorf("invalid seconds %q", ss)
	}
	millis, err := strconv.Atoi(ms)
	if err != nil {
		return 0, fmt.Errorf("invalid milliseconds %q", ms)
	}
	return float64(h*3600+m*60+s) + float64(millis)/1000.0, nil
}

func mergeTranscriptSegments(segments []models.TranscriptSegment, mergeGapSec float64, minFragmentSec float64) []models.TranscriptSegment {
	if len(segments) == 0 {
		return nil
	}

	curStart := segments[0].StartSec
	curEnd := segments[0].EndSec
	curText := segments[0].Text

	out := make([]models.TranscriptSegment, 0)

	flush := func() {
		if curEnd-curStart < minFragmentSec {
			return
		}
		out = append(out, models.TranscriptSegment{
			StartSec: curStart,
			EndSec:   curEnd,
			Text:     strings.TrimSpace(curText),
		})
	}

	for i := 1; i < len(segments); i++ {
		seg := segments[i]
		gap := seg.StartSec - curEnd
		if gap <= mergeGapSec {
			if seg.EndSec > curEnd {
				curEnd = seg.EndSec
			}
			curText = curText + " " + seg.Text
			continue
		}

		flush()
		curStart = seg.StartSec
		curEnd = seg.EndSec
		curText = seg.Text
	}

	flush()

	return out
}

// isWhisperMarker returns true for non-speech tags emitted by Whisper
func isWhisperMarker(text string) bool {
	t := strings.ToUpper(strings.TrimSpace(text))
	markers := []string{
		"[BLANK_AUDIO]",
		"[MUSIC]",
		"[SILENCE]",
		"[NOISE]",
		"[APPLAUSE]",
		"[LAUGHTER]",
		"(BLANK_AUDIO)",
		"(MUSIC)",
		"(SILENCE)",
	}
	for _, m := range markers {
		if t == m {
			return true
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}
