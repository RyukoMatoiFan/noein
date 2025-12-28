package video

import (
	"fmt"
	"noein/app/ffmpeg"
	"noein/app/models"
)

type Cutter struct {
	ffmpeg   *ffmpeg.FFmpegService
	videoMgr *VideoManager
}

func NewCutter(ffmpeg *ffmpeg.FFmpegService, videoMgr *VideoManager) *Cutter {
	return &Cutter{
		ffmpeg:   ffmpeg,
		videoMgr: videoMgr,
	}
}

// CutVideo exports a video segment from inFrame to outFrame
func (c *Cutter) CutVideo(videoID string, inFrame, outFrame int64, outputPath string, reEncode bool) error {
	return c.CutVideoWithCrop(videoID, inFrame, outFrame, outputPath, reEncode, nil)
}

// CutVideoWithCrop exports a video segment with optional cropping
func (c *Cutter) CutVideoWithCrop(videoID string, inFrame, outFrame int64, outputPath string, reEncode bool, crop *models.CropRegion) error {
	video, err := c.videoMgr.GetVideo(videoID)
	if err != nil {
		return err
	}

	// Validate frame range
	if inFrame < 0 || inFrame >= video.TotalFrames {
		return fmt.Errorf("inFrame %d out of range [0, %d)", inFrame, video.TotalFrames)
	}
	if outFrame < 0 || outFrame >= video.TotalFrames {
		return fmt.Errorf("outFrame %d out of range [0, %d)", outFrame, video.TotalFrames)
	}
	if inFrame >= outFrame {
		return fmt.Errorf("inFrame must be less than outFrame")
	}

	return c.ffmpeg.CutVideoWithCrop(video.Path, inFrame, outFrame, video.FrameRate, outputPath, reEncode, crop)
}

// CutVideoWithCropByPath exports a video segment with optional cropping using direct path
func (c *Cutter) CutVideoWithCropByPath(inputPath string, inFrame, outFrame int64, frameRate float64, outputPath string, reEncode bool, crop *models.CropRegion) error {
	return c.ffmpeg.CutVideoWithCrop(inputPath, inFrame, outFrame, frameRate, outputPath, reEncode, crop)
}

// ConcatVideos concatenates multiple videos using FFmpeg
func (c *Cutter) ConcatVideos(concatListPath, outputPath string) error {
	return c.ffmpeg.ConcatVideos(concatListPath, outputPath)
}

// CropVideoByPath crops a video using direct path (no time trimming)
func (c *Cutter) CropVideoByPath(inputPath, outputPath string, crop *models.CropRegion) error {
	return c.ffmpeg.CropVideo(inputPath, crop, outputPath)
}

// ApplyTransform applies a transform operation (scale, rotate, grayscale, frame_skip)
func (c *Cutter) ApplyTransform(inputPath, outputPath string, op *models.EditOperation) error {
	return c.ffmpeg.ApplyTransformOperation(inputPath, outputPath, op)
}
