package app

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

// VideoFileServer serves video files over HTTP for HTML5 video playback
type VideoFileServer struct {
	server   *http.Server
	port     int
	mu       sync.Mutex
	videoMap map[string]string // videoID -> file path
	ready    chan struct{}     // signals when server is ready
}

// NewVideoFileServer creates a new video file server
func NewVideoFileServer() *VideoFileServer {
	return &VideoFileServer{
		videoMap: make(map[string]string),
		ready:    make(chan struct{}),
	}
}

// Start starts the HTTP server on a random port
func (s *VideoFileServer) Start() error {
	// Find an available port
	listener, err := findAvailablePort(18456, 19000)
	if err != nil {
		close(s.ready) // Signal even on error
		return err
	}

	s.port = listener.Addr().(*net.TCPAddr).Port
	s.server = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.handleRequest(w, r)
		}),
	}

	// Signal that server is ready
	close(s.ready)

	// Start serving (blocking)
	err = s.server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// StartInBackground starts the server in a goroutine and waits for it to be ready
func (s *VideoFileServer) StartInBackground() error {
	go func() {
		s.Start()
	}()
	// Wait for server to be ready
	<-s.ready
	return nil
}

// findAvailablePort finds an available port in the given range
func findAvailablePort(start, end int) (net.Listener, error) {
	for port := start; port < end; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return listener, nil
		}
	}
	return nil, fmt.Errorf("could not find available port in range %d-%d", start, end)
}

// RegisterVideo registers a video file so it can be served
func (s *VideoFileServer) RegisterVideo(videoID, filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.videoMap[videoID] = filePath
}

// GetURL returns the URL for a video ID
func (s *VideoFileServer) GetURL(videoID string) string {
	return fmt.Sprintf("http://localhost:%d/video/%s", s.port, videoID)
}

// handleRequest handles incoming HTTP requests
func (s *VideoFileServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Extract video ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[1] != "video" {
		http.NotFound(w, r)
		return
	}

	videoID := parts[2]

	// Look up file path
	s.mu.Lock()
	filePath, ok := s.videoMap[videoID]
	s.mu.Unlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	// Set proper headers for video streaming
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".webm":
		w.Header().Set("Content-Type", "video/webm")
	case ".mov":
		w.Header().Set("Content-Type", "video/quicktime")
	case ".avi":
		w.Header().Set("Content-Type", "video/x-msvideo")
	case ".mkv":
		w.Header().Set("Content-Type", "video/x-matroska")
	default:
		w.Header().Set("Content-Type", "video/mp4")
	}

	// Enable range requests for seeking
	w.Header().Set("Accept-Ranges", "bytes")

	// Serve the file with range support
	http.ServeFile(w, r, filePath)
}

// Stop stops the server
func (s *VideoFileServer) Stop() {
	if s.server != nil {
		s.server.Close()
	}
}
