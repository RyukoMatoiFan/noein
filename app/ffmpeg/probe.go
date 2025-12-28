package ffmpeg

import (
	"encoding/json"
	"fmt"
	"noein/app/models"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/google/uuid"
)

type ProbeService struct {
	ffprobePath string
}

func NewProbeService(ffprobePath string) *ProbeService {
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	return &ProbeService{
		ffprobePath: ffprobePath,
	}
}

type FFProbeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"` // e.g. "30000/1001" for 29.97fps
		AvgFrameRate string `json:"avg_frame_rate"`
		BitRate    string `json:"bit_rate"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
}

func (p *ProbeService) GetVideoMetadata(videoPath string) (*models.VideoFile, error) {
	cmd := exec.Command(
		p.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)

	// Hide console window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probeData FFProbeOutput
	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Find video stream
	var videoStream *struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
		AvgFrameRate string `json:"avg_frame_rate"`
		BitRate    string `json:"bit_rate"`
	}

	for i := range probeData.Streams {
		if probeData.Streams[i].CodecType == "video" {
			videoStream = &probeData.Streams[i]
			break
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found in %s", videoPath)
	}

	// Parse duration
	duration, err := strconv.ParseFloat(probeData.Format.Duration, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}

	// Parse frame rate (handle fraction format like "30000/1001")
	frameRate, err := parseFrameRate(videoStream.AvgFrameRate)
	if err != nil {
		// Fallback to r_frame_rate
		frameRate, err = parseFrameRate(videoStream.RFrameRate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse frame rate: %w", err)
		}
	}

	// Calculate total frames
	totalFrames := int64(duration * frameRate)

	// Parse bit rate
	var bitRate int64
	if videoStream.BitRate != "" {
		bitRate, _ = strconv.ParseInt(videoStream.BitRate, 10, 64)
	} else if probeData.Format.BitRate != "" {
		bitRate, _ = strconv.ParseInt(probeData.Format.BitRate, 10, 64)
	}

	return &models.VideoFile{
		ID:          uuid.New().String(),
		Path:        videoPath,
		Name:        filepath.Base(videoPath),
		Duration:    duration,
		FrameRate:   frameRate,
		TotalFrames: totalFrames,
		Width:       videoStream.Width,
		Height:      videoStream.Height,
		Codec:       videoStream.CodecName,
		BitRate:     bitRate,
	}, nil
}

func parseFrameRate(frameRateStr string) (float64, error) {
	if frameRateStr == "" || frameRateStr == "0/0" {
		return 0, fmt.Errorf("invalid frame rate: %s", frameRateStr)
	}

	var num, den int
	_, err := fmt.Sscanf(frameRateStr, "%d/%d", &num, &den)
	if err != nil {
		// Try parsing as simple float
		return strconv.ParseFloat(frameRateStr, 64)
	}

	if den == 0 {
		return 0, fmt.Errorf("division by zero in frame rate: %s", frameRateStr)
	}

	return float64(num) / float64(den), nil
}
