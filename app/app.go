package app

import (
	"context"
	"encoding/json"
	"fmt"
	"noein/app/ffmpeg"
	"noein/app/models"
	"noein/app/video"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx            context.Context
	videoManager   *video.VideoManager
	frameExtractor *video.FrameExtractor
	cutter         *video.Cutter
	projectState   *models.ProjectState
	videoServer    *VideoFileServer
}

func NewApp() *App {
	return &App{
		projectState: &models.ProjectState{
			EditStack: make([]models.EditOperation, 0),
		},
	}
}

// Startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Start video file server
	a.videoServer = NewVideoFileServer()
	a.videoServer.StartInBackground()

	// Detect FFmpeg location (bundled or system)
	ffmpegPath := detectFFmpegPath()
	ffprobePath := detectFFprobePath()

	// Initialize FFmpeg services
	probe := ffmpeg.NewProbeService(ffprobePath)
	ffmpegSvc := ffmpeg.NewFFmpegService(ffmpegPath)

	// Initialize video services
	a.videoManager = video.NewVideoManager(probe)
	a.frameExtractor = video.NewFrameExtractor(ffmpegSvc, a.videoManager, 100)
	a.cutter = video.NewCutter(ffmpegSvc, a.videoManager)
}

// SelectFolder shows a folder selection dialog
func (a *App) SelectFolder() (string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Video Folder",
	})
	return folder, err
}

// GetVideoURL returns a URL for serving the video file
func (a *App) GetVideoURL(videoID string) (string, error) {
	video, err := a.videoManager.GetVideo(videoID)
	if err != nil {
		return "", err
	}

	// Register the video with the file server
	a.videoServer.RegisterVideo(videoID, video.Path)

	// Return the URL for the video
	return a.videoServer.GetURL(videoID), nil
}

// LoadVideoFolder scans a folder for video files and returns their metadata
func (a *App) LoadVideoFolder(folderPath string) ([]*models.VideoFile, error) {
	return a.videoManager.LoadFolder(folderPath)
}

// GetVideoMetadata returns metadata for a specific video (extracts if not already loaded)
func (a *App) GetVideoMetadata(videoID string) (*models.VideoFile, error) {
	return a.videoManager.GetVideo(videoID)
}

// SetCurrentVideo sets the current video ID, triggers metadata extraction if needed, and returns the video
func (a *App) SetCurrentVideo(videoID string) (*models.VideoFile, error) {
	a.projectState.CurrentVideoID = videoID

	// Trigger metadata extraction on-demand if not already loaded
	video, err := a.videoManager.GetVideo(videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to load video metadata: %w", err)
	}

	return video, nil
}

// GetFrame extracts a single frame from a video
func (a *App) GetFrame(videoID string, frameNumber int64) (*models.Frame, error) {
	return a.frameExtractor.GetFrame(videoID, frameNumber)
}

// GetFramePreview extracts 5 frames around the center frame
func (a *App) GetFramePreview(videoID string, centerFrame int64) (*models.FramePreview, error) {
	return a.frameExtractor.GetFramePreview(videoID, centerFrame)
}

// SetInPoint sets the in point for the current video
func (a *App) SetInPoint(frameNumber int64) error {
	a.projectState.InPoint = &frameNumber
	return nil
}

// SetOutPoint sets the out point for the current video
func (a *App) SetOutPoint(frameNumber int64) error {
	a.projectState.OutPoint = &frameNumber
	return nil
}

// ClearMarks clears the in/out points
func (a *App) ClearMarks() error {
	a.projectState.InPoint = nil
	a.projectState.OutPoint = nil
	return nil
}

// AddTrimExternal adds an external trim operation (keep only IN to OUT)
func (a *App) AddTrimExternal() error {
	if a.projectState.InPoint == nil || a.projectState.OutPoint == nil {
		return fmt.Errorf("both in and out points must be set")
	}

	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "trim_external",
		InFrame:     *a.projectState.InPoint,
		OutFrame:    *a.projectState.OutPoint,
		Description: fmt.Sprintf("Keep frames %d-%d", *a.projectState.InPoint, *a.projectState.OutPoint),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		// Remove operation if apply fails
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	// Clear current in/out after adding
	a.projectState.InPoint = nil
	a.projectState.OutPoint = nil

	return nil
}

// AddTrimInternal adds an internal trim operation (remove IN to OUT)
func (a *App) AddTrimInternal() error {
	if a.projectState.InPoint == nil || a.projectState.OutPoint == nil {
		return fmt.Errorf("both in and out points must be set")
	}

	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "trim_internal",
		InFrame:     *a.projectState.InPoint,
		OutFrame:    *a.projectState.OutPoint,
		Description: fmt.Sprintf("Remove frames %d-%d", *a.projectState.InPoint, *a.projectState.OutPoint),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		// Remove operation if apply fails
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	// Clear current in/out after adding
	a.projectState.InPoint = nil
	a.projectState.OutPoint = nil

	return nil
}

// AddCropOperation adds a crop operation to the edit stack
func (a *App) AddCropOperation() error {
	if a.projectState.CurrentCrop == nil {
		return fmt.Errorf("no crop region set")
	}

	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "crop",
		Crop:        a.projectState.CurrentCrop,
		Description: fmt.Sprintf("Crop to %dx%d at (%d,%d)", a.projectState.CurrentCrop.Width, a.projectState.CurrentCrop.Height, a.projectState.CurrentCrop.X, a.projectState.CurrentCrop.Y),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		// Remove operation if apply fails
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	// Clear current crop after adding
	a.projectState.CurrentCrop = nil

	return nil
}

// SetCropRegion sets the current crop region
func (a *App) SetCropRegion(x, y, width, height int) error {
	a.projectState.CurrentCrop = &models.CropRegion{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
	return nil
}

// ClearCropRegion clears the current crop region
func (a *App) ClearCropRegion() error {
	a.projectState.CurrentCrop = nil
	return nil
}

// AddScaleOperation adds a scale/resize operation to the edit stack
func (a *App) AddScaleOperation(width, height int) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid dimensions: width and height must be positive")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "scale",
		ScaleWidth:  width,
		ScaleHeight: height,
		Description: fmt.Sprintf("Scale to %dx%d", width, height),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddRotateOperation adds a rotation/flip operation to the edit stack
func (a *App) AddRotateOperation(rotateType string) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	// Validate rotate type
	validTypes := map[string]bool{
		"90": true, "180": true, "270": true,
		"hflip": true, "vflip": true,
	}
	if !validTypes[rotateType] {
		return fmt.Errorf("invalid rotate type: %s (must be 90, 180, 270, hflip, or vflip)", rotateType)
	}

	var description string
	switch rotateType {
	case "90":
		description = "Rotate 90° clockwise"
	case "180":
		description = "Rotate 180°"
	case "270":
		description = "Rotate 270° clockwise"
	case "hflip":
		description = "Flip horizontally"
	case "vflip":
		description = "Flip vertically"
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "rotate",
		RotateType:  rotateType,
		Description: description,
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddGrayscaleOperation adds a grayscale conversion operation to the edit stack
func (a *App) AddGrayscaleOperation() error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "grayscale",
		Description: "Convert to grayscale",
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddFrameSkipOperation adds a frame skip/downsample operation to the edit stack
func (a *App) AddFrameSkipOperation(skipN int) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	if skipN <= 1 {
		return fmt.Errorf("skipN must be greater than 1")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "frame_skip",
		FrameSkip:   skipN,
		Description: fmt.Sprintf("Extract every %d frame(s)", skipN),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddFPSChangeOperation adds a frame rate conversion operation to the edit stack
func (a *App) AddFPSChangeOperation(targetFPS float64) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	if targetFPS <= 0 || targetFPS > 240 {
		return fmt.Errorf("target FPS must be between 0 and 240")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "fps_change",
		TargetFPS:   targetFPS,
		Description: fmt.Sprintf("Change frame rate to %.2f fps", targetFPS),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddBrightnessContrastOperation adds a brightness/contrast adjustment operation to the edit stack
func (a *App) AddBrightnessContrastOperation(brightness, contrast float64) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	if brightness < -1.0 || brightness > 1.0 {
		return fmt.Errorf("brightness must be between -1.0 and 1.0")
	}

	if contrast < -1.0 || contrast > 1.0 {
		return fmt.Errorf("contrast must be between -1.0 and 1.0")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "brightness_contrast",
		Brightness:  brightness,
		Contrast:    contrast,
		Description: fmt.Sprintf("Adjust brightness: %.2f, contrast: %.2f", brightness, contrast),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddRemoveAudioOperation adds an audio removal operation to the edit stack
func (a *App) AddRemoveAudioOperation() error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "remove_audio",
		Description: "Remove audio track",
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddSpeedChangeOperation adds a speed adjustment operation to the edit stack
func (a *App) AddSpeedChangeOperation(speedFactor float64) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	if speedFactor <= 0 || speedFactor > 10 {
		return fmt.Errorf("speed factor must be between 0 and 10")
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "speed_change",
		SpeedFactor: speedFactor,
		Description: fmt.Sprintf("Change speed to %.2fx", speedFactor),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddPaddingOperation adds a padding/letterboxing operation to the edit stack
func (a *App) AddPaddingOperation(width, height int, color string) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	if width <= 0 || height <= 0 {
		return fmt.Errorf("padding dimensions must be positive")
	}

	if color == "" {
		color = "black"
	}

	operation := models.EditOperation{
		ID:            uuid.New().String(),
		Type:          "add_padding",
		PaddingWidth:  width,
		PaddingHeight: height,
		PaddingColor:  color,
		Description:   fmt.Sprintf("Add padding to %dx%d (%s)", width, height, color),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddTrimDurationOperation adds a duration-based trim operation to the edit stack
func (a *App) AddTrimDurationOperation(duration float64) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	if duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}

	operation := models.EditOperation{
		ID:           uuid.New().String(),
		Type:         "trim_duration",
		TrimDuration: duration,
		Description:  fmt.Sprintf("Keep first %.2f seconds", duration),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// AddFormatConversion adds a format conversion operation to the edit stack
func (a *App) AddFormatConversion(format, codec string) error {
	if a.projectState.CurrentVideoID == "" {
		return fmt.Errorf("no video selected")
	}

	// Validate format
	validFormats := map[string]bool{
		"mp4":  true,
		"avi":  true,
		"mkv":  true,
		"mov":  true,
		"webm": true,
	}
	if !validFormats[format] {
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Validate codec
	validCodecs := map[string]bool{
		"h264": true,
		"h265": true,
		"vp9":  true,
	}
	if !validCodecs[codec] {
		return fmt.Errorf("unsupported codec: %s", codec)
	}

	codecName := codec
	if codec == "h264" {
		codecName = "H.264"
	} else if codec == "h265" {
		codecName = "H.265/HEVC"
	} else if codec == "vp9" {
		codecName = "VP9"
	}

	operation := models.EditOperation{
		ID:          uuid.New().String(),
		Type:        "format_conversion",
		Format:      format,
		Codec:       codec,
		Description: fmt.Sprintf("Convert to %s (%s)", format, codecName),
	}

	a.projectState.EditStack = append(a.projectState.EditStack, operation)

	// Apply operation immediately
	if err := a.applyOperationToTemp(operation); err != nil {
		a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]
		return fmt.Errorf("failed to apply operation: %w", err)
	}

	return nil
}

// UndoLastEdit removes the last edit operation from the stack
func (a *App) UndoLastEdit() error {
	if len(a.projectState.EditStack) == 0 {
		return fmt.Errorf("no edits to undo")
	}

	// Remove last temp file if exists
	if len(a.projectState.TempFiles) > 0 {
		lastTemp := a.projectState.TempFiles[len(a.projectState.TempFiles)-1]
		os.Remove(lastTemp)
		a.projectState.TempFiles = a.projectState.TempFiles[:len(a.projectState.TempFiles)-1]
	}

	// Remove last operation
	a.projectState.EditStack = a.projectState.EditStack[:len(a.projectState.EditStack)-1]

	// Update current video
	if len(a.projectState.TempFiles) > 0 {
		// Use the previous temp file
		a.projectState.CurrentTempVideo = a.projectState.TempFiles[len(a.projectState.TempFiles)-1]

		// Load the previous temp file into video manager
		tempVideoID := "temp_" + uuid.New().String()[:8]
		err := a.videoManager.LoadTempVideo(tempVideoID, a.projectState.CurrentTempVideo)
		if err != nil {
			return fmt.Errorf("failed to load previous temp video: %w", err)
		}
		a.projectState.CurrentVideoID = tempVideoID
	} else {
		// Revert to original video
		a.projectState.CurrentTempVideo = ""
		if a.projectState.OriginalVideoID != "" {
			a.projectState.CurrentVideoID = a.projectState.OriginalVideoID
			a.projectState.OriginalVideoID = ""
		}
	}

	return nil
}

// ClearEditStack clears all edit operations
func (a *App) ClearEditStack() error {
	// Clean up all temp files
	for _, tempFile := range a.projectState.TempFiles {
		os.Remove(tempFile)
	}
	a.projectState.TempFiles = make([]string, 0)
	a.projectState.EditStack = make([]models.EditOperation, 0)

	// Revert to original video
	a.projectState.CurrentTempVideo = ""
	if a.projectState.OriginalVideoID != "" {
		a.projectState.CurrentVideoID = a.projectState.OriginalVideoID
		a.projectState.OriginalVideoID = ""
	}

	return nil
}

// applyOperationToTemp applies an operation to the current video and creates a temp file
func (a *App) applyOperationToTemp(op models.EditOperation) error {
	// Save original video ID on first operation
	if len(a.projectState.EditStack) == 1 && a.projectState.OriginalVideoID == "" {
		a.projectState.OriginalVideoID = a.projectState.CurrentVideoID
	}

	// Get current video (could be temp or original)
	var currentPath string
	var frameRate float64

	if a.projectState.CurrentTempVideo != "" {
		// Working with a temp file
		currentPath = a.projectState.CurrentTempVideo
		// Get frame rate from the temp video
		video, err := a.videoManager.GetVideo(a.projectState.CurrentVideoID)
		if err != nil {
			return err
		}
		frameRate = video.FrameRate
	} else {
		// Working with original
		video, err := a.videoManager.GetVideo(a.projectState.CurrentVideoID)
		if err != nil {
			return err
		}
		currentPath = video.Path
		frameRate = video.FrameRate
	}

	// Determine file extension for temp file
	tempExt := ".mp4" // Default extension
	if op.Type == "format_conversion" {
		tempExt = "." + op.Format
	} else if a.projectState.CurrentTempVideo != "" {
		// Preserve extension from previous temp file
		tempExt = filepath.Ext(a.projectState.CurrentTempVideo)
	}

	// Create temp output file
	tempDir := os.TempDir()
	tempOutput := filepath.Join(tempDir, fmt.Sprintf("noein_edit_%s%s", uuid.New().String()[:8], tempExt))

	// Apply the operation
	err := a.applyEditOperation(currentPath, tempOutput, op, frameRate)
	if err != nil {
		return err
	}

	// Add to temp files list
	a.projectState.TempFiles = append(a.projectState.TempFiles, tempOutput)
	a.projectState.CurrentTempVideo = tempOutput

	// Load the temp file as a new video
	tempVideoID := "temp_" + uuid.New().String()[:8]
	err = a.videoManager.LoadTempVideo(tempVideoID, tempOutput)
	if err != nil {
		return fmt.Errorf("failed to load temp video: %w", err)
	}

	// Update current video ID
	a.projectState.CurrentVideoID = tempVideoID

	return nil
}

// SaveToEditedFolder saves the edited video to "edited" subfolder
func (a *App) SaveToEditedFolder() (string, error) {
	if a.projectState.CurrentVideoID == "" {
		return "", fmt.Errorf("no video selected")
	}

	if len(a.projectState.EditStack) == 0 {
		return "", fmt.Errorf("no edits to apply")
	}

	// Get original video for naming
	originalVideoID := a.projectState.OriginalVideoID
	if originalVideoID == "" {
		originalVideoID = a.projectState.CurrentVideoID
	}

	originalVideo, err := a.videoManager.GetVideo(originalVideoID)
	if err != nil {
		return "", err
	}

	// Create "edited" subfolder in the same directory as the original video
	videoDir := filepath.Dir(originalVideo.Path)
	editedDir := filepath.Join(videoDir, "edited")
	if err := os.MkdirAll(editedDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create edited folder: %w", err)
	}

	// Generate unique output filename (prevent overwriting)
	baseName := filepath.Base(originalVideo.Name)
	ext := filepath.Ext(baseName)
	nameWithoutExt := baseName[:len(baseName)-len(ext)]

	// Check if there's a format conversion in the edit stack
	targetExt := ext
	for _, op := range a.projectState.EditStack {
		if op.Type == "format_conversion" {
			targetExt = "." + op.Format
			break
		}
	}

	// Find a unique filename by adding a counter if needed
	outputPath := filepath.Join(editedDir, fmt.Sprintf("%s_edited%s", nameWithoutExt, targetExt))
	counter := 1
	for {
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			// File doesn't exist, we can use this name
			break
		}
		// File exists, try next counter
		outputPath = filepath.Join(editedDir, fmt.Sprintf("%s_edited_%d%s", nameWithoutExt, counter, targetExt))
		counter++
	}

	// Since operations are already applied, just copy the final temp file
	if a.projectState.CurrentTempVideo != "" {
		// Copy temp file to final destination
		err = a.copyFile(a.projectState.CurrentTempVideo, outputPath)
		if err != nil {
			return "", fmt.Errorf("failed to save edited video: %w", err)
		}
	} else {
		return "", fmt.Errorf("no edited video to save")
	}

	// Clean up temp files after successful save
	for _, tempFile := range a.projectState.TempFiles {
		os.Remove(tempFile)
	}
	a.projectState.TempFiles = make([]string, 0)
	a.projectState.EditStack = make([]models.EditOperation, 0)
	a.projectState.CurrentTempVideo = ""

	// Revert to original video
	if a.projectState.OriginalVideoID != "" {
		a.projectState.CurrentVideoID = a.projectState.OriginalVideoID
		a.projectState.OriginalVideoID = ""
	}

	return outputPath, nil
}

// ApplyEditStackToVideos applies the current edit stack to multiple videos
func (a *App) ApplyEditStackToVideos(videoIDs []string) ([]models.BatchResult, error) {
	if len(a.projectState.EditStack) == 0 {
		return nil, fmt.Errorf("no edits to apply")
	}

	if len(videoIDs) == 0 {
		return nil, fmt.Errorf("no videos selected")
	}

	results := make([]models.BatchResult, len(videoIDs))

	// Check if there's a format conversion in the edit stack
	targetFormat := ""
	for _, op := range a.projectState.EditStack {
		if op.Type == "format_conversion" {
			targetFormat = op.Format
			break
		}
	}

	// Process each video
	for i, videoID := range videoIDs {
		result := models.BatchResult{VideoID: videoID}

		// Get video metadata
		video, err := a.videoManager.GetVideo(videoID)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to get video: %v", err)
			results[i] = result
			continue
		}

		// Create "edited" subfolder in the same directory as the original video
		videoDir := filepath.Dir(video.Path)
		editedDir := filepath.Join(videoDir, "edited")
		if err := os.MkdirAll(editedDir, 0755); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create edited folder: %v", err)
			results[i] = result
			continue
		}

		// Generate unique output filename (prevent overwriting)
		baseName := filepath.Base(video.Name)
		ext := filepath.Ext(baseName)
		nameWithoutExt := baseName[:len(baseName)-len(ext)]

		// Use target format extension if format conversion is in the stack
		targetExt := ext
		if targetFormat != "" {
			targetExt = "." + targetFormat
		}

		// Find a unique filename by adding a counter if needed
		outputPath := filepath.Join(editedDir, fmt.Sprintf("%s_edited%s", nameWithoutExt, targetExt))
		counter := 1
		for {
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				// File doesn't exist, we can use this name
				break
			}
			// File exists, try next counter
			outputPath = filepath.Join(editedDir, fmt.Sprintf("%s_edited_%d%s", nameWithoutExt, counter, targetExt))
			counter++
		}

		// Apply all operations in sequence
		currentInput := video.Path
		var tempFiles []string

		// Create temp directory for this video's processing
		tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("noein_batch_%s_%s", uuid.New().String()[:8], videoID))
		os.MkdirAll(tempDir, 0755)
		defer os.RemoveAll(tempDir)

		success := true
		currentExt := ext
		for j, op := range a.projectState.EditStack {
			// Update extension if this is a format conversion operation
			if op.Type == "format_conversion" {
				currentExt = "." + op.Format
			}
			tempOutput := filepath.Join(tempDir, fmt.Sprintf("op_%d_%s%s", j, videoID, currentExt))

			err := a.applyEditOperation(currentInput, tempOutput, op, video.FrameRate)
			if err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("operation %d (%s) failed: %v", j+1, op.Type, err)
				success = false
				break
			}

			tempFiles = append(tempFiles, tempOutput)
			currentInput = tempOutput
		}

		if !success {
			// Clean up temp files for this video
			for _, tf := range tempFiles {
				os.Remove(tf)
			}
			results[i] = result
			continue
		}

		// Move final result to output location
		err = a.copyFile(currentInput, outputPath)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to save output: %v", err)
		} else {
			result.Success = true
			result.Output = outputPath
		}

		// Clean up temp files for this video
		for _, tf := range tempFiles {
			os.Remove(tf)
		}

		results[i] = result
	}

	return results, nil
}

// SavePanelStates saves UI panel states to config file
func (a *App) SavePanelStates(panelStates map[string]bool) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "noein", "panel_states.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Convert to JSON and save
	data, err := json.MarshalIndent(panelStates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal panel states: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadPanelStates loads UI panel states from config file
func (a *App) LoadPanelStates() (map[string]bool, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "noein", "panel_states.json")

	// If config doesn't exist, return default (all collapsed)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return map[string]bool{
			"cut":            false,
			"history":        false,
			"crop":           false,
			"transform":      false,
			"frameOps":       false,
			"adjustments":    false,
			"advanced":       false,
			"format":         false,
			"fileManagement": false,
			"info":           false,
		}, nil
	}

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var panelStates map[string]bool
	if err := json.Unmarshal(data, &panelStates); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return panelStates, nil
}

// copyFile copies a file from src to dst
func (a *App) copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

// applyEditOperation applies a single edit operation
func (a *App) applyEditOperation(inputPath, outputPath string, op models.EditOperation, frameRate float64) error {
	switch op.Type {
	case "trim_external":
		// Keep only frames from InFrame to OutFrame
		return a.cutter.CutVideoWithCropByPath(inputPath, op.InFrame, op.OutFrame, frameRate, outputPath, true, nil)

	case "trim_internal":
		// Remove frames from InFrame to OutFrame (keep before and after)
		return a.applyInternalTrim(inputPath, outputPath, op.InFrame, op.OutFrame, frameRate)

	case "crop":
		// Apply crop to entire video (no time trimming)
		return a.cutter.CropVideoByPath(inputPath, outputPath, op.Crop)

	case "scale", "rotate", "grayscale", "frame_skip", "fps_change", "brightness_contrast", "remove_audio", "speed_change", "add_padding", "trim_duration", "format_conversion":
		// Apply transform operations
		return a.cutter.ApplyTransform(inputPath, outputPath, &op)

	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// applyInternalTrim removes a section from the middle of a video
func (a *App) applyInternalTrim(inputPath, outputPath string, inFrame, outFrame int64, frameRate float64) error {
	// This requires splitting the video into two parts and concatenating them
	// Part 1: 0 to inFrame
	// Part 2: outFrame to end

	video, err := a.videoManager.GetVideo(a.projectState.CurrentVideoID)
	if err != nil {
		return err
	}

	// Create temp files for the two parts
	tempDir := os.TempDir()
	part1Path := filepath.Join(tempDir, fmt.Sprintf("noein_part1_%s.mp4", uuid.New().String()[:8]))
	part2Path := filepath.Join(tempDir, fmt.Sprintf("noein_part2_%s.mp4", uuid.New().String()[:8]))
	defer os.Remove(part1Path)
	defer os.Remove(part2Path)

	// Extract part 1 (before the cut)
	if inFrame > 0 {
		err = a.cutter.CutVideoWithCropByPath(inputPath, 0, inFrame-1, frameRate, part1Path, true, nil)
		if err != nil {
			return fmt.Errorf("failed to extract part 1: %w", err)
		}
	}

	// Extract part 2 (after the cut)
	if outFrame < video.TotalFrames {
		err = a.cutter.CutVideoWithCropByPath(inputPath, outFrame+1, video.TotalFrames-1, frameRate, part2Path, true, nil)
		if err != nil {
			return fmt.Errorf("failed to extract part 2: %w", err)
		}
	}

	// Concatenate the two parts
	// Create concat file list
	concatListPath := filepath.Join(tempDir, fmt.Sprintf("noein_concat_%s.txt", uuid.New().String()[:8]))
	defer os.Remove(concatListPath)

	var concatContent string
	if inFrame > 0 {
		concatContent += fmt.Sprintf("file '%s'\n", part1Path)
	}
	if outFrame < video.TotalFrames {
		concatContent += fmt.Sprintf("file '%s'\n", part2Path)
	}

	if err := os.WriteFile(concatListPath, []byte(concatContent), 0644); err != nil {
		return fmt.Errorf("failed to create concat list: %w", err)
	}

	// Use FFmpeg concat demuxer
	return a.cutter.ConcatVideos(concatListPath, outputPath)
}

// GetProjectState returns the current project state
func (a *App) GetProjectState() *models.ProjectState {
	return a.projectState
}

// SelectOutputFile shows a save file dialog
func (a *App) SelectOutputFile() (string, error) {
	file, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Exported Video",
		DefaultFilename: "output.mp4",
		Filters: []runtime.FileFilter{
			{DisplayName: "MP4 Video", Pattern: "*.mp4"},
		},
	})
	return file, err
}

// SelectOutputDirectory shows a directory selection dialog for batch export
func (a *App) SelectOutputDirectory() (string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Output Directory for Batch Export",
	})
	return folder, err
}

// ExportSegment exports a video segment from inFrame to outFrame
func (a *App) ExportSegment(videoID string, inFrame, outFrame int64, outputPath string) error {
	// Default to re-encode for frame-perfect cutting
	return a.cutter.CutVideo(videoID, inFrame, outFrame, outputPath, true)
}

// detectFFmpegPath detects the location of ffmpeg.exe
func detectFFmpegPath() string {
	// Check if ffmpeg is bundled with the application
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		bundledPath := filepath.Join(exeDir, "ffmpeg.exe")
		if _, err := os.Stat(bundledPath); err == nil {
			return bundledPath
		}
	}

	// Fall back to system PATH
	return "ffmpeg"
}

// detectFFprobePath detects the location of ffprobe.exe
func detectFFprobePath() string {
	// Check if ffprobe is bundled with the application
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		bundledPath := filepath.Join(exeDir, "ffprobe.exe")
		if _, err := os.Stat(bundledPath); err == nil {
			return bundledPath
		}
	}

	// Fall back to system PATH
	return "ffprobe"
}

// DeleteVideoFile deletes a video file from disk
func (a *App) DeleteVideoFile(videoID string) error {
	video, err := a.videoManager.GetVideo(videoID)
	if err != nil {
		return fmt.Errorf("failed to get video: %w", err)
	}

	// Delete the file
	if err := os.Remove(video.Path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Remove from video manager
	a.videoManager.RemoveVideo(videoID)

	// If this was the current video, clear it
	if a.projectState.CurrentVideoID == videoID {
		a.projectState.CurrentVideoID = ""
	}

	return nil
}

// MoveVideoToFolder moves a video file to a specified folder
func (a *App) MoveVideoToFolder(videoID, targetFolder string) (string, error) {
	video, err := a.videoManager.GetVideo(videoID)
	if err != nil {
		return "", fmt.Errorf("failed to get video: %w", err)
	}

	// Create target folder if it doesn't exist
	if err := os.MkdirAll(targetFolder, 0755); err != nil {
		return "", fmt.Errorf("failed to create target folder: %w", err)
	}

	// Build new file path
	fileName := filepath.Base(video.Path)
	newPath := filepath.Join(targetFolder, fileName)

	// Check if file already exists at destination
	if _, err := os.Stat(newPath); err == nil {
		// File exists, add counter to make it unique
		ext := filepath.Ext(fileName)
		nameWithoutExt := fileName[:len(fileName)-len(ext)]
		counter := 1
		for {
			newPath = filepath.Join(targetFolder, fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext))
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				break
			}
			counter++
		}
	}

	// Move the file
	if err := os.Rename(video.Path, newPath); err != nil {
		return "", fmt.Errorf("failed to move file: %w", err)
	}

	// Update video manager with new path
	a.videoManager.UpdateVideoPath(videoID, newPath)

	return newPath, nil
}
