package speech

import (
	"encoding/json"
	"fmt"
	"noein/app/models"
	"strings"
)

func RenderTranscript(format string, segments []models.TranscriptSegment) ([]byte, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "srt":
		return []byte(renderSRT(segments)), "srt", nil
	case "vtt":
		return []byte(renderVTT(segments)), "vtt", nil
	case "json":
		data, err := json.MarshalIndent(segments, "", "  ")
		if err != nil {
			return nil, "", err
		}
		return data, "json", nil
	default:
		return nil, "", fmt.Errorf("unsupported transcript format: %s", format)
	}
}

func renderSRT(segments []models.TranscriptSegment) string {
	var b strings.Builder
	for i, s := range segments {
		if s.EndSec <= s.StartSec {
			continue
		}
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%d\n", i+1))
		b.WriteString(fmt.Sprintf("%s --> %s\n", formatSRTTime(s.StartSec), formatSRTTime(s.EndSec)))
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	return b.String()
}

func renderVTT(segments []models.TranscriptSegment) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for _, s := range segments {
		if s.EndSec <= s.StartSec {
			continue
		}
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s --> %s\n", formatVTTTime(s.StartSec), formatVTTTime(s.EndSec)))
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	return b.String()
}

func formatSRTTime(sec float64) string {
	hh, mm, ss, ms := splitTime(sec)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hh, mm, ss, ms)
}

func formatVTTTime(sec float64) string {
	hh, mm, ss, ms := splitTime(sec)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hh, mm, ss, ms)
}

func splitTime(sec float64) (hh int, mm int, ss int, ms int) {
	if sec < 0 {
		sec = 0
	}
	totalMillis := int64(sec * 1000.0)
	hh = int(totalMillis / (3600 * 1000))
	totalMillis -= int64(hh) * 3600 * 1000
	mm = int(totalMillis / (60 * 1000))
	totalMillis -= int64(mm) * 60 * 1000
	ss = int(totalMillis / 1000)
	totalMillis -= int64(ss) * 1000
	ms = int(totalMillis)
	if ms < 0 {
		ms = 0
	}
	return hh, mm, ss, ms
}
