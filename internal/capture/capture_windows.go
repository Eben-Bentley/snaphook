//go:build windows

package capture

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/kbinani/screenshot"
	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	procGetCursorPos = user32.NewProc("GetCursorPos")
)

type POINT struct {
	X, Y int32
}

func getCursorPosition() (int, int, error) {
	var pt POINT
	ret, _, err := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return 0, 0, err
	}
	return int(pt.X), int(pt.Y), nil
}

func getDisplayAtCursor() int {
	x, y, err := getCursorPosition()
	if err != nil {
		return 0
	}

	n := screenshot.NumActiveDisplays()
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		if x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y {
			return i
		}
	}

	return 0
}

func captureScreen() (string, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return "", fmt.Errorf("no active displays found")
	}

	displayIndex := getDisplayAtCursor()
	bounds := screenshot.GetDisplayBounds(displayIndex)

	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", fmt.Errorf("screenshot capture failed: %w", err)
	}

	// Save to temp file
	tmpDir := os.TempDir()
	imagePath := filepath.Join(tmpDir, fmt.Sprintf("snapview-%d.png", time.Now().UnixNano()))

	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	if autoSaveEnabled && autoSaveDir != "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		permanentPath := filepath.Join(autoSaveDir, fmt.Sprintf("screenshot_%s.png", timestamp))

		permFile, err := os.Create(permanentPath)
		if err == nil {
			defer permFile.Close()

			file.Seek(0, 0)
			permFile.ReadFrom(file)
		}
	}

	return imagePath, nil
}
