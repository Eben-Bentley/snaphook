//go:build windows

package clipboard

import (
	"fmt"
	"os/exec"
	"syscall"
)

func copyImage(imagePath string) error {
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$image = [System.Drawing.Image]::FromFile('%s')
[System.Windows.Forms.Clipboard]::SetImage($image)
$image.Dispose()
`, imagePath)

	cmd := exec.Command("powershell", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return nil
}
