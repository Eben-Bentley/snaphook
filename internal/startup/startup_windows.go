//go:build windows

package startup

import (
	"os"
	"os/exec"
	"path/filepath"
)

func GetStartupPath() string {
	appData := os.Getenv("APPDATA")
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "SnapView.lnk")
}

func IsEnabled() bool {
	_, err := os.Stat(GetStartupPath())
	return err == nil
}

func Enable() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	startupPath := GetStartupPath()

	script := `$WshShell = New-Object -ComObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut("` + startupPath + `"); $Shortcut.TargetPath = "` + exePath + `"; $Shortcut.Save()`

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	return cmd.Run()
}

func Disable() error {
	return os.Remove(GetStartupPath())
}
