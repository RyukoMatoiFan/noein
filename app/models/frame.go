package models

type Frame struct {
	FrameNumber int64   `json:"frameNumber"`
	Timestamp   float64 `json:"timestamp"` // seconds
	ImageData   string  `json:"imageData"` // base64 encoded PNG
}

type FramePreview struct {
	CenterFrame int64   `json:"centerFrame"`
	Frames      []Frame `json:"frames"` // 5 frames: [-2, -1, 0, +1, +2]
}
