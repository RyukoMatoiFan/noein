package models

type SpeechFragment struct {
	ID          string   `json:"id"`
	StartSec    float64  `json:"startSec"`
	EndSec      float64  `json:"endSec"`
	InFrame     int64    `json:"inFrame"`
	OutFrame    int64    `json:"outFrame"`
	Text        string   `json:"text"`
	TextEnglish string   `json:"textEnglish,omitempty"`
	Label       string   `json:"label,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type TranscriptSegment struct {
	StartSec    float64 `json:"startSec"`
	EndSec      float64 `json:"endSec"`
	Text        string  `json:"text"`
	TextEnglish string  `json:"textEnglish,omitempty"`
}
