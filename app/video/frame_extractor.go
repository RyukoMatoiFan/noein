package video

import (
	"encoding/base64"
	"fmt"
	"noein/app/ffmpeg"
	"noein/app/models"
	"sync"
)

type FrameCache struct {
	cache   map[string]*models.Frame
	keys    []string
	maxSize int
	mu      sync.RWMutex
}

func NewFrameCache(maxSize int) *FrameCache {
	return &FrameCache{
		cache:   make(map[string]*models.Frame),
		keys:    make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

func (fc *FrameCache) Get(videoID string, frameNumber int64) *models.Frame {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", videoID, frameNumber)
	return fc.cache[key]
}

func (fc *FrameCache) Put(videoID string, frameNumber int64, frame *models.Frame) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	key := fmt.Sprintf("%s:%d", videoID, frameNumber)

	// If key already exists, remove it from keys list
	for i, k := range fc.keys {
		if k == key {
			fc.keys = append(fc.keys[:i], fc.keys[i+1:]...)
			break
		}
	}

	// Add new key to end (most recent)
	fc.keys = append(fc.keys, key)
	fc.cache[key] = frame

	// Evict oldest if over capacity
	if len(fc.keys) > fc.maxSize {
		oldestKey := fc.keys[0]
		fc.keys = fc.keys[1:]
		delete(fc.cache, oldestKey)
	}
}

type FrameExtractor struct {
	ffmpeg   *ffmpeg.FFmpegService
	cache    *FrameCache
	videoMgr *VideoManager
}

func NewFrameExtractor(ffmpeg *ffmpeg.FFmpegService, videoMgr *VideoManager, cacheSize int) *FrameExtractor {
	if cacheSize <= 0 {
		cacheSize = 100
	}

	return &FrameExtractor{
		ffmpeg:   ffmpeg,
		cache:    NewFrameCache(cacheSize),
		videoMgr: videoMgr,
	}
}

// GetFrame extracts a single frame from a video
func (fe *FrameExtractor) GetFrame(videoID string, frameNumber int64) (*models.Frame, error) {
	// Check cache first
	if cached := fe.cache.Get(videoID, frameNumber); cached != nil {
		return cached, nil
	}

	// Get video metadata
	video, err := fe.videoMgr.GetVideo(videoID)
	if err != nil {
		return nil, err
	}

	// Validate frame number
	if frameNumber < 0 || frameNumber >= video.TotalFrames {
		return nil, fmt.Errorf("frame number %d out of range [0, %d)", frameNumber, video.TotalFrames)
	}

	// Extract frame
	imageData, err := fe.ffmpeg.ExtractFrame(video.Path, frameNumber, video.FrameRate)
	if err != nil {
		return nil, err
	}

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	frame := &models.Frame{
		FrameNumber: frameNumber,
		Timestamp:   float64(frameNumber) / video.FrameRate,
		ImageData:   "data:image/png;base64," + base64Data,
	}

	// Cache the frame
	fe.cache.Put(videoID, frameNumber, frame)

	return frame, nil
}

// GetFramePreview extracts 5 frames around the center frame (center ± 2)
func (fe *FrameExtractor) GetFramePreview(videoID string, centerFrame int64) (*models.FramePreview, error) {
	video, err := fe.videoMgr.GetVideo(videoID)
	if err != nil {
		return nil, err
	}

	frames := make([]models.Frame, 5)
	var wg sync.WaitGroup
	errors := make(chan error, 5)
	frameMutex := sync.Mutex{}

	// Extract 5 frames in parallel
	for i := -2; i <= 2; i++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()

			frameNum := centerFrame + int64(offset)
			index := offset + 2

			// Handle out of bounds frames
			if frameNum < 0 || frameNum >= video.TotalFrames {
				frameMutex.Lock()
				frames[index] = models.Frame{
					FrameNumber: frameNum,
					Timestamp:   float64(frameNum) / video.FrameRate,
					ImageData:   "", // Empty frame
				}
				frameMutex.Unlock()
				return
			}

			frame, err := fe.GetFrame(videoID, frameNum)
			if err != nil {
				errors <- err
				return
			}

			frameMutex.Lock()
			frames[index] = *frame
			frameMutex.Unlock()
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check if there were any errors
	if len(errors) > 0 {
		return nil, <-errors
	}

	return &models.FramePreview{
		CenterFrame: centerFrame,
		Frames:      frames,
	}, nil
}
