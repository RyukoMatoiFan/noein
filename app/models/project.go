package models

type CropRegion struct {
	X      int `json:"x"`      // X position (pixels from left)
	Y      int `json:"y"`      // Y position (pixels from top)
	Width  int `json:"width"`  // Width of crop region
	Height int `json:"height"` // Height of crop region
}

// EditOperation represents a single non-destructive edit operation
type EditOperation struct {
	ID            string      `json:"id"`
	Type          string      `json:"type"`        // "trim_external", "trim_internal", "crop", "frame_skip", "scale", "rotate", "grayscale", "fps_change", "brightness_contrast", "remove_audio", "speed_change", "add_padding", "trim_duration", "format_conversion"
	InFrame       int64       `json:"inFrame"`     // For trim operations
	OutFrame      int64       `json:"outFrame"`    // For trim operations
	Crop          *CropRegion `json:"crop,omitempty"`     // For crop operations
	FrameSkip     int         `json:"frameSkip,omitempty"` // For frame_skip: extract every Nth frame
	ScaleWidth    int         `json:"scaleWidth,omitempty"`  // For scale: target width
	ScaleHeight   int         `json:"scaleHeight,omitempty"` // For scale: target height
	RotateType    string      `json:"rotateType,omitempty"`  // For rotate: "90", "180", "270", "hflip", "vflip"
	TargetFPS     float64     `json:"targetFps,omitempty"`   // For fps_change: target frame rate
	Brightness    float64     `json:"brightness,omitempty"`  // For brightness_contrast: -1.0 to 1.0
	Contrast      float64     `json:"contrast,omitempty"`    // For brightness_contrast: -1.0 to 1.0
	SpeedFactor   float64     `json:"speedFactor,omitempty"` // For speed_change: 0.5 = half speed, 2.0 = double speed
	PaddingWidth  int         `json:"paddingWidth,omitempty"`  // For add_padding: target width with padding
	PaddingHeight int         `json:"paddingHeight,omitempty"` // For add_padding: target height with padding
	PaddingColor  string      `json:"paddingColor,omitempty"`  // For add_padding: color (e.g., "black", "white")
	TrimDuration  float64     `json:"trimDuration,omitempty"`  // For trim_duration: duration in seconds
	Format        string      `json:"format,omitempty"`  // For format_conversion: output format (mp4, avi, mkv, mov, webm)
	Codec         string      `json:"codec,omitempty"`   // For format_conversion: output codec (h264, h265, vp9)
	Description   string      `json:"description"` // User-friendly description
}

type ProjectState struct {
	CurrentVideoID   string          `json:"currentVideoId"`
	CurrentFrame     int64           `json:"currentFrame"`
	InPoint          *int64          `json:"inPoint"`
	OutPoint         *int64          `json:"outPoint"`
	CurrentCrop      *CropRegion     `json:"currentCrop,omitempty"`     // Current crop selection
	EditStack        []EditOperation `json:"editStack"`                 // Stack of edit operations
	TempFiles        []string        `json:"-"`                         // Temp files (not serialized)
	CurrentTempVideo string          `json:"currentTempVideo,omitempty"` // Current temp video being worked on
	OriginalVideoID  string          `json:"originalVideoId,omitempty"`  // Original video before edits
}

// BatchResult represents the result of applying operations to a single video
type BatchResult struct {
	VideoID string `json:"videoId"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Output  string `json:"output,omitempty"`
}
