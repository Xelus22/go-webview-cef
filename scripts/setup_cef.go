package main

import (
	"archive/tar"
	"compress/bzip2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	CEFVersion = "145.0.26+g6ed7554+chromium-145.0.7632.110" // CEF 145 - Chromium 145.0.7632.110
)

func main() {
	platform := detectPlatform()
	arch := detectArch()

	fmt.Printf("Detected: %s/%s\n", platform, arch)

	if platform == "unsupported" || arch == "unsupported" {
		fmt.Println("Unsupported platform/architecture")
		os.Exit(1)
	}

	// Create target directory
	targetDir := filepath.Join("third_party", "cef", fmt.Sprintf("%s_%s", platform, arch))
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	// Check if already downloaded
	markerFile := filepath.Join(targetDir, ".cef_installed")
	if _, err := os.Stat(markerFile); err == nil {
		fmt.Printf("CEF already installed at: %s\n", targetDir)
		printEnvHelp(targetDir)
		return
	}

	// Build download URL
	url := buildDownloadURL(platform, arch)
	fmt.Printf("Downloading from: %s\n", url)

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("cef_download_%s.tar.bz2", CEFVersion))

	// Download
	if err := downloadFile(url, tempFile); err != nil {
		fmt.Printf("Download failed: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempFile)

	fmt.Println("Extracting...")

	// Extract
	if err := extractTarBz2(tempFile, targetDir); err != nil {
		fmt.Printf("Extraction failed: %v\n", err)
		os.Exit(1)
	}

	// Create marker file
	if err := os.WriteFile(markerFile, []byte(targetDir), 0644); err != nil {
		fmt.Printf("Failed to create marker file: %v\n", err)
	}

	// Create environment marker
	envFile := filepath.Join(targetDir, "cef_env.txt")
	envContent := fmt.Sprintf("CEF_ROOT=%s\n", targetDir)
	os.WriteFile(envFile, []byte(envContent), 0644)

	fmt.Printf("CEF installed successfully at: %s\n", targetDir)
	printEnvHelp(targetDir)
}

func detectPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "macos"
	case "linux":
		return "linux"
	default:
		return "unsupported"
	}
}

func detectArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "64"
	case "arm64":
		return "arm64"
	default:
		return "unsupported"
	}
}

func buildDownloadURL(platform, arch string) string {
	var cefPlatform string
	switch platform {
	case "windows":
		if arch == "64" {
			cefPlatform = "windows64"
		} else {
			cefPlatform = "windowsarm64"
		}
	case "macos":
		if arch == "64" {
			cefPlatform = "macosx64"
		} else {
			cefPlatform = "macosarm64"
		}
	case "linux":
		if arch == "64" {
			cefPlatform = "linux64"
		} else {
			cefPlatform = "linuxarm64"
		}
	}

	return fmt.Sprintf("https://cef-builds.spotifycdn.com/cef_binary_%s_%s.tar.bz2",
		CEFVersion, cefPlatform)
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarBz2(src, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	bz2Reader := bzip2.NewReader(file)
	tarReader := tar.NewReader(bz2Reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dst, stripTopDir(header.Name))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

func stripTopDir(path string) string {
	parts := strings.SplitN(path, string(filepath.Separator), 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return path
}

func printEnvHelp(targetDir string) {
	fmt.Println("\n=== Environment Setup ===")
	fmt.Printf("Set CEF_ROOT environment variable:\n")
	fmt.Printf("  export CEF_ROOT=%s\n", targetDir)
	fmt.Println("\nOr use the environment file:")
	fmt.Printf("  source %s/cef_env.txt\n", targetDir)
}
