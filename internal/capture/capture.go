package capture

import (
	"runtime"
	"sync"
)

var (
	autoSaveEnabled bool
	autoSaveDir     string
	autoSaveMutex   sync.RWMutex
)

func SetAutoSave(enabled bool, dir string) {
	autoSaveMutex.Lock()
	defer autoSaveMutex.Unlock()
	autoSaveEnabled = enabled
	autoSaveDir = dir
}

func getAutoSaveConfig() (bool, string) {
	autoSaveMutex.RLock()
	defer autoSaveMutex.RUnlock()
	return autoSaveEnabled, autoSaveDir
}

func CaptureScreen() (string, error) {
	return captureScreen()
}

func init() {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		return
	}
	panic("unsupported platform: " + runtime.GOOS)
}
