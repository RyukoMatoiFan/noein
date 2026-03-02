package ffmpeg

import (
	"bytes"
	"fmt"
	"noein/app/models"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"
)

type FFmpegService struct {
	ffmpegPath string
}

func NewFFmpegService(ffmpegPath string) *FFmpegService {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &FFmpegService{
		ffmpegPath: ffmpegPath,
	}
}

func (f *FFmpegService) ExtractAudioWav(inputPath string, outputWavPath string) error {
	cmd := exec.Command(
		f.ffmpegPath,
		"-y",
		"-i", inputPath,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-c:a", "pcm_s16le",
		outputWavPath,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg audio extract failed: %w (stderr: %s)", err, errBuf.String())
	}

	return nil
}

// SilencePeriod represents a detected silence interval in the audio.
type SilencePeriod struct {
	StartSec float64
	EndSec   float64
}

// DetectSilences runs FFmpeg silencedetect filter and returns silence periods.
func (f *FFmpegService) DetectSilences(inputPath string, minSilenceDurationSec float64, silenceThresholdDb int) ([]SilencePeriod, error) {
	if minSilenceDurationSec <= 0 {
		minSilenceDurationSec = 0.3
	}
	if silenceThresholdDb >= 0 {
		silenceThresholdDb = -50
	}

	af := fmt.Sprintf("silencedetect=noise=%ddB:d=%.3f", silenceThresholdDb, minSilenceDurationSec)
	cmd := exec.Command(
		f.ffmpegPath,
		"-i", inputPath,
		"-af", af,
		"-f", "null",
		"-",
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg silencedetect failed: %w (stderr: %s)", err, errBuf.String())
	}

	// silencedetect outputs to stderr
	output := errBuf.String()
	return parseSilenceDetectOutput(output), nil
}

var silenceStartRE = regexp.MustCompile(`silence_start:\s*([\d.]+)`)
var silenceEndRE = regexp.MustCompile(`silence_end:\s*([\d.]+)`)

func parseSilenceDetectOutput(output string) []SilencePeriod {
	starts := silenceStartRE.FindAllStringSubmatch(output, -1)
	ends := silenceEndRE.FindAllStringSubmatch(output, -1)

	var periods []SilencePeriod
	for i := 0; i < len(starts) && i < len(ends); i++ {
		s, err1 := strconv.ParseFloat(starts[i][1], 64)
		e, err2 := strconv.ParseFloat(ends[i][1], 64)
		if err1 == nil && err2 == nil && e > s {
			periods = append(periods, SilencePeriod{StartSec: s, EndSec: e})
		}
	}
	return periods
}

// CutAudio extracts an audio segment by time range. Output format is inferred from outputPath extension.
func (f *FFmpegService) CutAudio(inputPath string, startSec, endSec float64, outputPath string) error {
	args := []string{
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.6f", startSec),
		"-to", fmt.Sprintf("%.6f", endSec),
		"-vn",
		"-c:a", "pcm_s16le",
		"-ac", "1",
		"-ar", "16000",
		"-y",
		outputPath,
	}

	cmd := exec.Command(f.ffmpegPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg audio cut failed: %w (stderr: %s)", err, errBuf.String())
	}
	return nil
}

// ExtractFrame extracts a single frame at the specified frame number
func (f *FFmpegService) ExtractFrame(videoPath string, frameNumber int64, frameRate float64) ([]byte, error) {
	// Calculate timestamp from frame number
	timestamp := float64(frameNumber) / frameRate

	// Use -ss before -i for faster seeking
	cmd := exec.Command(
		f.ffmpegPath,
		"-ss", fmt.Sprintf("%.6f", timestamp),
		"-i", videoPath,
		"-vframes", "1",
		"-f", "image2pipe",
		"-vcodec", "png",
		"-",
	)

	// Hide console window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w (stderr: %s)", err, errBuf.String())
	}

	return outBuf.Bytes(), nil
}

// CutVideo cuts a video segment from inFrame to outFrame
func (f *FFmpegService) CutVideo(videoPath string, inFrame, outFrame int64, frameRate float64, outputPath string, reEncode bool) error {
	return f.CutVideoWithCrop(videoPath, inFrame, outFrame, frameRate, outputPath, reEncode, nil)
}

// CutVideoWithCrop cuts a video segment with optional cropping
func (f *FFmpegService) CutVideoWithCrop(videoPath string, inFrame, outFrame int64, frameRate float64, outputPath string, reEncode bool, crop *models.CropRegion) error {
	inTime := float64(inFrame) / frameRate
	outTime := float64(outFrame) / frameRate

	var args []string

	if reEncode {
		// Build video filter
		var vf string
		if crop != nil {
			// Add crop filter: crop=width:height:x:y
			vf = fmt.Sprintf("crop=%d:%d:%d:%d", crop.Width, crop.Height, crop.X, crop.Y)
		}

		// Frame-perfect cutting with re-encoding
		args = []string{
			"-i", videoPath,
			"-ss", fmt.Sprintf("%.6f", inTime),
			"-to", fmt.Sprintf("%.6f", outTime),
		}

		if vf != "" {
			args = append(args, "-vf", vf)
		}

		args = append(args,
			"-c:v", "libx264",
			"-crf", "18",
			"-preset", "slow",
			"-c:a", "copy",
			"-y", // Overwrite output file
			outputPath,
		)
	} else {
		if crop != nil {
			// Cannot use stream copy with crop - must re-encode
			return fmt.Errorf("crop requires re-encoding; reEncode must be true")
		}

		// Fast stream copy (not frame-perfect)
		args = []string{
			"-ss", fmt.Sprintf("%.6f", inTime),
			"-i", videoPath,
			"-to", fmt.Sprintf("%.6f", outTime-inTime),
			"-c", "copy",
			"-y",
			outputPath,
		}
	}

	cmd := exec.Command(f.ffmpegPath, args...)

	// Hide console window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg cut failed: %w (stderr: %s)", err, errBuf.String())
	}

	return nil
}

// ConcatVideos concatenates multiple videos using FFmpeg concat demuxer
func (f *FFmpegService) ConcatVideos(concatListPath, outputPath string) error {
	cmd := exec.Command(
		f.ffmpegPath,
		"-f", "concat",
		"-safe", "0",
		"-i", concatListPath,
		"-c", "copy",
		"-y",
		outputPath,
	)

	// Hide console window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg concat failed: %w (stderr: %s)", err, errBuf.String())
	}

	return nil
}

// CropVideo crops a video without time-based trimming
func (f *FFmpegService) CropVideo(videoPath string, crop *models.CropRegion, outputPath string) error {
	// Apply crop filter to entire video
	vf := fmt.Sprintf("crop=%d:%d:%d:%d", crop.Width, crop.Height, crop.X, crop.Y)

	args := []string{
		"-i", videoPath,
		"-vf", vf,
		"-c:v", "libx264",
		"-crf", "18",
		"-preset", "slow",
		"-c:a", "copy",
		"-y",
		outputPath,
	}

	cmd := exec.Command(f.ffmpegPath, args...)

	// Hide console window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg crop failed: %w (stderr: %s)", err, errBuf.String())
	}

	return nil
}

// ApplyTransformOperation applies a transform operation (scale, rotate, grayscale, frame_skip, etc.)
func (f *FFmpegService) ApplyTransformOperation(inputPath, outputPath string, op *models.EditOperation) error {
	var args []string
	var vf string // Video filter string

	switch op.Type {
	case "scale":
		// Scale to specific resolution: scale=1280:720
		vf = fmt.Sprintf("scale=%d:%d", op.ScaleWidth, op.ScaleHeight)
		args = []string{"-i", inputPath, "-vf", vf, "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "copy", "-y", outputPath}

	case "rotate":
		// Rotation and flips
		switch op.RotateType {
		case "90":
			vf = "transpose=1" // 90 degrees clockwise
		case "180":
			vf = "transpose=1,transpose=1" // 180 degrees
		case "270":
			vf = "transpose=2" // 90 degrees counter-clockwise
		case "hflip":
			vf = "hflip" // Horizontal flip
		case "vflip":
			vf = "vflip" // Vertical flip
		default:
			return fmt.Errorf("unknown rotate type: %s", op.RotateType)
		}
		args = []string{"-i", inputPath, "-vf", vf, "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "copy", "-y", outputPath}

	case "grayscale":
		// Convert to grayscale
		vf = "format=gray"
		args = []string{"-i", inputPath, "-vf", vf, "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "copy", "-y", outputPath}

	case "frame_skip":
		// Extract every Nth frame: select='not(mod(n,N))'
		// Use -vsync vfr to maintain frame timestamps
		vf = fmt.Sprintf("select='not(mod(n,%d))'", op.FrameSkip)
		args = []string{"-i", inputPath, "-vf", vf, "-vsync", "vfr", "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "copy", "-y", outputPath}

	case "fps_change":
		// Change frame rate: fps=30
		vf = fmt.Sprintf("fps=%.3f", op.TargetFPS)
		args = []string{"-i", inputPath, "-vf", vf, "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "copy", "-y", outputPath}

	case "brightness_contrast":
		// Adjust brightness and contrast: eq=brightness=0.06:contrast=1.2
		// Brightness: -1.0 (black) to 1.0 (white), 0 = no change
		// Contrast: -1000 to 1000, 1.0 = no change
		contrastValue := 1.0 + op.Contrast // Convert -1 to 1 range to 0 to 2 for FFmpeg
		vf = fmt.Sprintf("eq=brightness=%.3f:contrast=%.3f", op.Brightness, contrastValue)
		args = []string{"-i", inputPath, "-vf", vf, "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "copy", "-y", outputPath}

	case "remove_audio":
		// Remove audio stream: -an flag
		args = []string{"-i", inputPath, "-c:v", "copy", "-an", "-y", outputPath}

	case "speed_change":
		// Change video speed: setpts for video, atempo for audio
		// Speed up: setpts=0.5*PTS (2x speed), atempo=2.0
		// Slow down: setpts=2.0*PTS (0.5x speed), atempo=0.5
		ptsMultiplier := 1.0 / op.SpeedFactor
		vf = fmt.Sprintf("setpts=%.3f*PTS", ptsMultiplier)

		// FFmpeg atempo filter has limits: 0.5 to 2.0
		// For larger changes, chain multiple atempo filters
		audioFilter := ""
		if op.SpeedFactor >= 0.5 && op.SpeedFactor <= 2.0 {
			audioFilter = fmt.Sprintf("atempo=%.3f", op.SpeedFactor)
		} else if op.SpeedFactor > 2.0 {
			// Chain multiple atempo=2.0 filters
			audioFilter = "atempo=2.0"
			remaining := op.SpeedFactor / 2.0
			for remaining > 2.0 {
				audioFilter += ",atempo=2.0"
				remaining /= 2.0
			}
			if remaining > 1.0 {
				audioFilter += fmt.Sprintf(",atempo=%.3f", remaining)
			}
		} else if op.SpeedFactor < 0.5 {
			// Chain multiple atempo=0.5 filters
			audioFilter = "atempo=0.5"
			remaining := op.SpeedFactor / 0.5
			for remaining < 0.5 {
				audioFilter += ",atempo=0.5"
				remaining /= 0.5
			}
			if remaining < 1.0 {
				audioFilter += fmt.Sprintf(",atempo=%.3f", remaining)
			}
		}

		args = []string{"-i", inputPath, "-vf", vf, "-af", audioFilter, "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-y", outputPath}

	case "add_padding":
		// Add padding/letterboxing: pad=width:height:x:y:color
		// Center the video: x=(ow-iw)/2, y=(oh-ih)/2
		color := op.PaddingColor
		if color == "" {
			color = "black"
		}
		vf = fmt.Sprintf("pad=%d:%d:(ow-iw)/2:(oh-ih)/2:%s", op.PaddingWidth, op.PaddingHeight, color)
		args = []string{"-i", inputPath, "-vf", vf, "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "copy", "-y", outputPath}

	case "trim_duration":
		// Trim to specific duration from start: -t duration
		args = []string{"-i", inputPath, "-t", fmt.Sprintf("%.3f", op.TrimDuration), "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "copy", "-y", outputPath}

	case "format_conversion":
		// Convert to different format and codec
		var codec string
		switch op.Codec {
		case "h264":
			codec = "libx264"
		case "h265":
			codec = "libx265"
		case "vp9":
			codec = "libvpx-vp9"
		default:
			return fmt.Errorf("unsupported codec: %s", op.Codec)
		}

		// Build args for format conversion
		args = []string{
			"-i", inputPath,
			"-c:v", codec,
			"-crf", "18",
			"-preset", "slow",
			"-c:a", "copy",
			"-y",
			outputPath,
		}

	default:
		return fmt.Errorf("unsupported transform operation type: %s", op.Type)
	}

	cmd := exec.Command(f.ffmpegPath, args...)

	// Hide console window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg transform failed: %w (stderr: %s)", err, errBuf.String())
	}

	return nil
}
