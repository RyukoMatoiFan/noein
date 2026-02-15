package speech

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"noein/app/runtimepaths"
)

func WhisperInstallDir() (string, error) {
	base, err := runtimepaths.NoeinRuntimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "whisper"), nil
}

func EnsureWhisperCLI() (string, error) {
	baseDir, err := WhisperInstallDir()
	if err != nil {
		return "", err
	}

	installDir := filepath.Join(baseDir, "whisper-bin-x64")
	targetExe := filepath.Join(installDir, "Release", "whisper-cli.exe")
	if _, err := os.Stat(targetExe); err == nil {
		return targetExe, nil
	}

	assetURL, err := githubLatestAssetURL("ggml-org", "whisper.cpp", "whisper-bin-x64.zip")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create whisper install dir: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "noein_whisper_bin_*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := downloadFile(assetURL, tmpPath); err != nil {
		return "", err
	}

	if err := unzipToDir(tmpPath, installDir); err != nil {
		return "", err
	}

	if _, err := os.Stat(targetExe); err == nil {
		return targetExe, nil
	}

	found, err := findFileByBaseName(installDir, "whisper-cli.exe")
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("whisper-cli.exe not found after extraction")
	}
	return found, nil
}

func EnsureWhisperModel(modelName string) (string, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		modelName = "tiny.en-q5_1"
	}

	baseDir, err := WhisperInstallDir()
	if err != nil {
		return "", err
	}

	modelsDir := filepath.Join(baseDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create models dir: %w", err)
	}

	fileName := modelName
	if !strings.HasSuffix(strings.ToLower(fileName), ".bin") {
		if strings.HasPrefix(fileName, "ggml-") {
			fileName = fileName + ".bin"
		} else {
			fileName = "ggml-" + fileName + ".bin"
		}
	}

	destPath := filepath.Join(modelsDir, fileName)
	if _, err := os.Stat(destPath); err == nil {
		return destPath, nil
	}

	url := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/" + fileName
	if err := downloadFile(url, destPath); err != nil {
		return "", err
	}

	return destPath, nil
}

func downloadFile(url string, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "noein")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: http %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func unzipToDir(zipPath string, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		name := f.Name
		name = strings.ReplaceAll(name, "\\", "/")
		name = strings.TrimPrefix(name, "/")
		clean := filepath.Clean(filepath.FromSlash(name))
		if clean == "." {
			continue
		}

		outPath := filepath.Join(destDir, clean)
		if !strings.HasPrefix(outPath, destDir+string(os.PathSeparator)) && outPath != destDir {
			return fmt.Errorf("invalid zip path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, 0755); err != nil {
				return fmt.Errorf("failed to create dir: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("failed to create dir: %w", err)
		}

		in, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry: %w", err)
		}

		out, err := os.Create(outPath)
		if err != nil {
			_ = in.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		_ = in.Close()
		if copyErr != nil {
			return fmt.Errorf("failed to extract file: %w", copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close file: %w", closeErr)
		}
	}

	return nil
}

func findFileByBaseName(rootDir string, baseName string) (string, error) {
	var found string
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), baseName) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && err != filepath.SkipAll {
		return "", err
	}
	return found, nil
}

type githubRelease struct {
	Assets []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func githubLatestAssetURL(owner string, repo string, assetName string) (string, error) {
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequest(http.MethodGet, api, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "noein")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch github release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("failed to fetch github release: http %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("failed to parse github release json: %w", err)
	}

	for _, a := range rel.Assets {
		if a.Name == assetName && a.BrowserDownloadURL != "" {
			return a.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("github release asset not found: %s", assetName)
}
