package video

import (
	"fmt"
	"noein/app/ffmpeg"
	"noein/app/models"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type VideoManager struct {
	probe  *ffmpeg.ProbeService
	videos map[string]*models.VideoFile
	mu     sync.RWMutex
}

func NewVideoManager(probe *ffmpeg.ProbeService) *VideoManager {
	return &VideoManager{
		probe:  probe,
		videos: make(map[string]*models.VideoFile),
	}
}

// LoadFolder scans a folder for video files and extracts all metadata concurrently
func (vm *VideoManager) LoadFolder(folderPath string) ([]*models.VideoFile, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	supportedExtensions := map[string]bool{
		".mp4":  true,
		".mov":  true,
		".avi":  true,
		".mkv":  true,
		".webm": true,
		".mp3":  true,
		".wav":  true,
		".flac": true,
		".aac":  true,
		".m4a":  true,
		".ogg":  true,
	}

	// Collect video paths
	var videoPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !supportedExtensions[ext] {
			continue
		}

		videoPath := filepath.Join(folderPath, entry.Name())
		videoPaths = append(videoPaths, videoPath)
	}

	// Extract metadata concurrently
	type result struct {
		video *models.VideoFile
		err   error
		index int
	}

	resultChan := make(chan result, len(videoPaths))
	var wg sync.WaitGroup

	// Limit concurrency to avoid overwhelming the system
	semaphore := make(chan struct{}, 4) // Process 4 videos at a time

	for i, videoPath := range videoPaths {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()

			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			video, err := vm.probe.GetVideoMetadata(path)
			if err != nil {
				// Even if metadata extraction fails, create a basic entry
				video = &models.VideoFile{
					ID:   generateIDFromPath(path),
					Path: path,
					Name: filepath.Base(path),
				}
			} else {
				video.ID = generateIDFromPath(path)
			}

			resultChan <- result{video: video, err: err, index: index}
		}(i, videoPath)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results in original order
	results := make([]*models.VideoFile, len(videoPaths))
	for res := range resultChan {
		results[res.index] = res.video
	}

	// Cache all videos
	vm.mu.Lock()
	for _, video := range results {
		vm.videos[video.ID] = video
	}
	vm.mu.Unlock()

	return results, nil
}

// LoadFile loads a single video file and returns it as a slice
func (vm *VideoManager) LoadFile(filePath string) ([]*models.VideoFile, error) {
	video, err := vm.probe.GetVideoMetadata(filePath)
	if err != nil {
		video = &models.VideoFile{
			ID:   generateIDFromPath(filePath),
			Path: filePath,
			Name: filepath.Base(filePath),
		}
	} else {
		video.ID = generateIDFromPath(filePath)
	}

	vm.mu.Lock()
	vm.videos[video.ID] = video
	vm.mu.Unlock()

	return []*models.VideoFile{video}, nil
}

// generateIDFromPath creates a simple ID from file path
func generateIDFromPath(path string) string {
	// Use a simple hash of the path as ID
	h := 0
	for i := 0; i < len(path); i++ {
		h = 31*h + int(path[i])
	}
	return fmt.Sprintf("%x", h)
}

// GetVideo retrieves a video by ID (metadata already loaded from LoadFolder)
func (vm *VideoManager) GetVideo(id string) (*models.VideoFile, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	video, ok := vm.videos[id]
	if !ok {
		return nil, fmt.Errorf("video not found: %s", id)
	}

	return video, nil
}


// LoadTempVideo loads a temporary video file into the manager
func (vm *VideoManager) LoadTempVideo(id string, videoPath string) error {
	video, err := vm.probe.GetVideoMetadata(videoPath)
	if err != nil {
		return fmt.Errorf("failed to probe temp video: %w", err)
	}

	// Override the ID with the provided temp ID
	video.ID = id

	vm.mu.Lock()
	vm.videos[id] = video
	vm.mu.Unlock()

	return nil
}

// RemoveVideo removes a video from the manager
func (vm *VideoManager) RemoveVideo(id string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	delete(vm.videos, id)
}

// UpdateVideoPath updates the path of a video in the manager
func (vm *VideoManager) UpdateVideoPath(id, newPath string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	video, ok := vm.videos[id]
	if !ok {
		return fmt.Errorf("video not found: %s", id)
	}

	video.Path = newPath
	video.Name = filepath.Base(newPath)
	return nil
}
