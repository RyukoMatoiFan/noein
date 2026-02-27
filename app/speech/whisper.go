package speech

import (
	"bytes"
	"fmt"
	"log"
	"noein/app/ffmpeg"
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

func (w *WhisperRunner) DetectSpeechFragments(
	inputVideoPath string,
	frameRate float64,
	videoDuration float64,
	mergeGapMs int,
	minFragmentMs int,
	splitOnSilence bool,
	silenceDurationMs int,
	silenceThresholdDb int,
	extractAudio func(inputPath, outputWavPath string) error,
	detectSilences func(inputPath string, minDurSec float64, threshDb int) ([]ffmpeg.SilencePeriod, error),
) ([]models.SpeechFragment, error) {
	if mergeGapMs <= 0 {
		mergeGapMs = 400
	}
	if minFragmentMs < 0 {
		minFragmentMs = 0
	}

	log.Printf("[SPEECH] === DetectSpeechFragments called === splitOnSilence=%v silenceDurationMs=%d silenceThresholdDb=%d mergeGapMs=%d minFragmentMs=%d",
		splitOnSilence, silenceDurationMs, silenceThresholdDb, mergeGapMs, minFragmentMs)

	// Run Whisper transcription
	segments, err := w.TranscribeSegments(inputVideoPath, extractAudio)
	if err != nil {
		return nil, err
	}
	log.Printf("[SPEECH] Whisper returned %d raw segments", len(segments))

	var merged []models.TranscriptSegment

	if splitOnSilence && detectSilences != nil {
		// Silence-first mode: derive speech regions from FFmpeg silence detection,
		// then assign Whisper text to each region. This works regardless of whether
		// Whisper properly detects the language.
		if silenceDurationMs <= 0 {
			silenceDurationMs = 300
		}
		if silenceThresholdDb >= 0 {
			silenceThresholdDb = -30
		}
		silences, sErr := detectSilences(inputVideoPath, float64(silenceDurationMs)/1000.0, silenceThresholdDb)
		if sErr != nil {
			log.Printf("[SPEECH] silence detection failed, falling back to Whisper-only: %v", sErr)
			merged = mergeTranscriptSegments(segments, float64(mergeGapMs)/1000.0, float64(minFragmentMs)/1000.0, nil)
		} else {
			log.Printf("[SPEECH] detected %d silence periods", len(silences))
			for i, sil := range silences {
				log.Printf("[SPEECH] silence %d: %.3f → %.3f (%.3fs)", i, sil.StartSec, sil.EndSec, sil.EndSec-sil.StartSec)
			}
			merged = speechRegionsFromSilences(silences, segments, videoDuration, float64(minFragmentMs)/1000.0)
			log.Printf("[SPEECH] silence-based splitting produced %d fragments", len(merged))
		}
	} else {
		// Whisper-only mode: merge based on gaps between Whisper segments.
		merged = mergeTranscriptSegments(segments, float64(mergeGapMs)/1000.0, float64(minFragmentMs)/1000.0, nil)
	}
	log.Printf("[SPEECH] final: %d fragments", len(merged))

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

func mergeTranscriptSegments(segments []models.TranscriptSegment, mergeGapSec float64, minFragmentSec float64, silences []ffmpeg.SilencePeriod) []models.TranscriptSegment {
	if len(segments) == 0 {
		return nil
	}

	// If silence detection is active, first split any Whisper segment that
	// contains a silence period inside it, then merge as usual.
	if len(silences) > 0 {
		for i, sil := range silences {
			log.Printf("[SILENCE] period %d: %.3f → %.3f (duration %.3fs)", i, sil.StartSec, sil.EndSec, sil.EndSec-sil.StartSec)
		}
		before := len(segments)
		segments = splitRawSegmentsAtSilences(segments, silences)
		log.Printf("[SILENCE] split %d whisper segments into %d segments", before, len(segments))
		for i, seg := range segments {
			log.Printf("[SILENCE] post-split segment %d: %.3f → %.3f text=%q", i, seg.StartSec, seg.EndSec, seg.Text)
		}
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

		// Check if any silence period falls in the gap between segments
		silenceInGap := false
		for _, sil := range silences {
			if sil.EndSec > curEnd-0.05 && sil.StartSec < seg.StartSec+0.05 {
				silenceInGap = true
				break
			}
		}

		if gap <= mergeGapSec && !silenceInGap {
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

// splitRawSegmentsAtSilences takes Whisper segments and silence periods, and splits
// any segment that contains a silence fully inside it. The text stays with the
// sub-segment before the silence; the sub-segment after gets "(cont.)" as text since
// we cannot know where the word boundary is within a single Whisper segment.
func splitRawSegmentsAtSilences(segments []models.TranscriptSegment, silences []ffmpeg.SilencePeriod) []models.TranscriptSegment {
	var result []models.TranscriptSegment

	for _, seg := range segments {
		// Find silences that overlap with the interior of this segment
		var inside []ffmpeg.SilencePeriod
		for _, sil := range silences {
			// Silence must start after the segment starts and end before the segment ends
			if sil.StartSec >= seg.StartSec && sil.EndSec <= seg.EndSec {
				inside = append(inside, sil)
			} else if sil.StartSec > seg.StartSec && sil.StartSec < seg.EndSec && sil.EndSec > seg.EndSec {
				// Silence starts inside but extends past end — clip it
				inside = append(inside, ffmpeg.SilencePeriod{StartSec: sil.StartSec, EndSec: seg.EndSec})
			}
		}

		if len(inside) == 0 {
			result = append(result, seg)
			continue
		}

		// Split segment around each silence: keep text with part before first silence
		curStart := seg.StartSec
		for _, sil := range inside {
			if sil.StartSec > curStart+0.01 {
				result = append(result, models.TranscriptSegment{
					StartSec: curStart,
					EndSec:   sil.StartSec,
					Text:     seg.Text,
				})
				// Only the first part gets the original text
				seg.Text = ""
			}
			curStart = sil.EndSec
		}
		// Remaining tail
		if seg.EndSec > curStart+0.01 {
			result = append(result, models.TranscriptSegment{
				StartSec: curStart,
				EndSec:   seg.EndSec,
				Text:     seg.Text,
			})
		}
	}

	return result
}

// speechRegionsFromSilences derives speech fragments by inverting silence periods.
// Everything between silences is a speech region. Whisper text is then matched to
// each region by overlap. This works regardless of Whisper's language support.
func speechRegionsFromSilences(silences []ffmpeg.SilencePeriod, whisperSegments []models.TranscriptSegment, videoDuration float64, minFragmentSec float64) []models.TranscriptSegment {
	if len(silences) == 0 && len(whisperSegments) == 0 {
		return nil
	}

	// Use actual video duration for the time range
	timeStart := 0.0
	timeEnd := videoDuration

	log.Printf("[SPEECH] inverting silences over range [%.3f, %.3f]", timeStart, timeEnd)

	// Invert silences to get speech regions
	var regions []models.TranscriptSegment
	cursor := timeStart
	for _, sil := range silences {
		if sil.StartSec > cursor+0.01 {
			// Gap before this silence = speech region
			regions = append(regions, models.TranscriptSegment{
				StartSec: cursor,
				EndSec:   sil.StartSec,
			})
		}
		if sil.EndSec > cursor {
			cursor = sil.EndSec
		}
	}
	// Trailing speech after last silence
	if timeEnd > cursor+0.01 {
		regions = append(regions, models.TranscriptSegment{
			StartSec: cursor,
			EndSec:   timeEnd,
		})
	}

	log.Printf("[SPEECH] inverted into %d raw speech regions", len(regions))
	for i, r := range regions {
		log.Printf("[SPEECH]   region %d: %.3f → %.3f (%.3fs)", i, r.StartSec, r.EndSec, r.EndSec-r.StartSec)
	}

	// Assign Whisper text to each region by finding overlapping segments
	for i := range regions {
		var texts []string
		for _, ws := range whisperSegments {
			overlapStart := ws.StartSec
			if regions[i].StartSec > overlapStart {
				overlapStart = regions[i].StartSec
			}
			overlapEnd := ws.EndSec
			if regions[i].EndSec < overlapEnd {
				overlapEnd = regions[i].EndSec
			}
			if overlapEnd > overlapStart+0.01 {
				text := strings.TrimSpace(ws.Text)
				if text != "" {
					texts = append(texts, text)
				}
			}
		}
		regions[i].Text = strings.Join(texts, " ")
	}

	// Filter by minimum duration
	var out []models.TranscriptSegment
	for _, r := range regions {
		if r.EndSec-r.StartSec >= minFragmentSec {
			out = append(out, r)
		}
	}

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
