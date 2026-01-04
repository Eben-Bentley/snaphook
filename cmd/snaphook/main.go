package main

import (
	"log"
	"os"
	"sync"
	"syscall"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"

	"snaphook/internal/assets"
	"snaphook/internal/capture"
	"snaphook/internal/clipboard"
	"snaphook/internal/config"
	"snaphook/internal/hotkey"
	"snaphook/internal/preview"
	"snaphook/internal/startup"
)

var (
	screenshotMutex      sync.Mutex
	screenshotInProgress bool
	currentConfig        *config.Config
	configMutex          sync.RWMutex
	instanceMutex        windows.Handle
)

func main() {
	mutexName, err := syscall.UTF16PtrFromString("Global\\SnapHook-SingleInstance-Mutex")
	if err != nil {
		log.Fatalf("Failed to create mutex name: %v", err)
	}

	instanceMutex, err = windows.CreateMutex(nil, false, mutexName)
	if err != nil {
		lastErr := windows.GetLastError()
		if lastErr == windows.ERROR_ALREADY_EXISTS || err.Error() == "Cannot create a file when that file already exists." {
			log.Println("Another instance of SnapHook is already running")
			os.Exit(1)
		}
		log.Fatalf("Failed to create mutex: %v", err)
	}

	defer windows.CloseHandle(instanceMutex)

	systray.Run(onReady, onExit)
}

func onReady() {
	var err error
	currentConfig, err = config.Load()
	if err != nil {
		log.Printf("Failed to load config: %v, using defaults", err)
		currentConfig = &config.Config{Hotkey: "Ctrl+Shift+S"}
	}

	capture.CleanupOldTempFiles()

	systray.SetIcon(assets.IconData)
	systray.SetTitle("SnapHook")
	configMutex.RLock()
	hotkeyStr := currentConfig.Hotkey
	enablePreview := currentConfig.EnablePreview
	copyToClipboard := currentConfig.CopyToClipboard
	autoSave := currentConfig.AutoSave
	configMutex.RUnlock()

	systray.SetTooltip("SnapHook - Press " + hotkeyStr + " to capture")

	mHotkey := systray.AddMenuItem("Hotkey: "+hotkeyStr, "Current screenshot hotkey")
	mHotkey.Disable()
	systray.AddSeparator()

	mViewPreview := systray.AddMenuItem("View Preview", "Open preview window in browser")
	mEnablePreview := systray.AddMenuItemCheckbox("Enable Preview", "Enable browser preview for screenshots", enablePreview)
	systray.AddSeparator()

	mCopyClipboard := systray.AddMenuItemCheckbox("Copy to Clipboard", "Copy screenshot to clipboard", copyToClipboard)
	mAutoSave := systray.AddMenuItemCheckbox("Auto-Save", "Save screenshots to Pictures/SnapHook", autoSave)
	systray.AddSeparator()

	mStartup := systray.AddMenuItemCheckbox("Start on Boot", "Start SnapView when Windows starts", startup.IsEnabled())
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit SnapView")

	if err := hotkey.Register(hotkeyStr, handleScreenshot); err != nil {
		log.Printf("Warning: Failed to register hotkey: %v", err)
		log.Println("The application will still run, but you'll need to manually configure the hotkey in your system settings.")
	}

	if autoSave {
		if err := config.EnsureAutoSaveDir(); err != nil {
			log.Printf("Failed to create auto-save directory: %v", err)
		} else {
			capture.SetAutoSave(true, config.GetAutoSaveDir())
		}
	}

	if enablePreview {
		preview.Start()
	} else {
		mViewPreview.Disable()
	}

	go func() {
		for {
			select {
			case <-mViewPreview.ClickedCh:
				preview.OpenBrowser()
			case <-mCopyClipboard.ClickedCh:
				configMutex.Lock()
				if mCopyClipboard.Checked() {
					currentConfig.CopyToClipboard = false
					mCopyClipboard.Uncheck()
				} else {
					currentConfig.CopyToClipboard = true
					mCopyClipboard.Check()
				}
				if err := config.Save(currentConfig); err != nil {
					log.Printf("Failed to save config: %v", err)
				}
				configMutex.Unlock()
			case <-mAutoSave.ClickedCh:
				configMutex.Lock()
				if mAutoSave.Checked() {
					currentConfig.AutoSave = false
					capture.SetAutoSave(false, "")
					mAutoSave.Uncheck()
				} else {
					if err := config.EnsureAutoSaveDir(); err != nil {
						log.Printf("Failed to create auto-save directory: %v", err)
					} else {
						currentConfig.AutoSave = true
						capture.SetAutoSave(true, config.GetAutoSaveDir())
						mAutoSave.Check()
					}
				}
				if err := config.Save(currentConfig); err != nil {
					log.Printf("Failed to save config: %v", err)
				}
				configMutex.Unlock()
			case <-mEnablePreview.ClickedCh:
				configMutex.Lock()
				if mEnablePreview.Checked() {
					currentConfig.EnablePreview = false
					mEnablePreview.Uncheck()
					preview.Shutdown()
					mViewPreview.Disable()
				} else {
					currentConfig.EnablePreview = true
					mEnablePreview.Check()
					preview.Start()
					mViewPreview.Enable()
				}
				if err := config.Save(currentConfig); err != nil {
					log.Printf("Failed to save config: %v", err)
				}
				configMutex.Unlock()
			case <-mStartup.ClickedCh:
				if mStartup.Checked() {
					if err := startup.Disable(); err != nil {
						log.Printf("Failed to disable startup: %v", err)
					} else {
						mStartup.Uncheck()
					}
				} else {
					if err := startup.Enable(); err != nil {
						log.Printf("Failed to enable startup: %v", err)
					} else {
						mStartup.Check()
					}
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	preview.Shutdown()
	hotkey.Unregister()
}

func handleScreenshot() {
	log.Println("Hotkey pressed - handleScreenshot called")

	screenshotMutex.Lock()
	if screenshotInProgress {
		log.Println("Screenshot already in progress, skipping")
		screenshotMutex.Unlock()
		return
	}
	screenshotInProgress = true
	screenshotMutex.Unlock()

	log.Println("Starting screenshot capture")

	go func() {
		imagePath, err := capture.CaptureScreen()
		if err != nil {
			log.Printf("Error capturing screen: %v", err)
			screenshotMutex.Lock()
			screenshotInProgress = false
			screenshotMutex.Unlock()
			return
		}
		log.Printf("Screenshot saved to: %s", imagePath)

		screenshotMutex.Lock()
		screenshotInProgress = false
		screenshotMutex.Unlock()
		log.Println("Screenshot captured - ready for next screenshot")

		configMutex.RLock()
		copyToClipboard := currentConfig.CopyToClipboard
		enablePreview := currentConfig.EnablePreview
		configMutex.RUnlock()

		if copyToClipboard {
			go clipboard.CopyImage(imagePath)
		}
		if enablePreview {
			preview.Show(imagePath)
		}
	}()
}
