package models

type VideoFile struct {
	ID          string  `json:"id"`
	Path        string  `json:"path"`
	Name        string  `json:"name"`
	Duration    float64 `json:"duration"`     // seconds
	FrameRate   float64 `json:"frameRate"`    // fps
	TotalFrames int64   `json:"totalFrames"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Codec       string  `json:"codec"`
	BitRate     int64   `json:"bitRate"`
	AudioOnly   bool    `json:"audioOnly"`
}
