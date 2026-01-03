package capture

import "runtime"

var autoSaveEnabled bool
var autoSaveDir string

func SetAutoSave(enabled bool, dir string) {
	autoSaveEnabled = enabled
	autoSaveDir = dir
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
