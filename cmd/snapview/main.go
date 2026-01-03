package main

import (
	"log"
	"sync"

	"github.com/getlantern/systray"

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
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	var err error
	currentConfig, err = config.Load()
	if err != nil {
		log.Printf("Failed to load config: %v, using defaults", err)
		currentConfig = &config.Config{Hotkey: "Ctrl+Shift+S"}
	}

	systray.SetTitle("SnapHook")
	systray.SetTooltip("SnapHook - Press " + currentConfig.Hotkey + " to capture")

	mHotkey := systray.AddMenuItem("Hotkey: "+currentConfig.Hotkey, "Current screenshot hotkey")
	mHotkey.Disable()
	systray.AddSeparator()

	mViewPreview := systray.AddMenuItem("View Preview", "Open preview window in browser")
	systray.AddSeparator()

	mCopyClipboard := systray.AddMenuItemCheckbox("Copy to Clipboard", "Copy screenshot to clipboard", currentConfig.CopyToClipboard)
	mAutoSave := systray.AddMenuItemCheckbox("Auto-Save", "Save screenshots to Pictures/SnapHook", currentConfig.AutoSave)
	systray.AddSeparator()

	mStartup := systray.AddMenuItemCheckbox("Start on Boot", "Start SnapView when Windows starts", startup.IsEnabled())
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit SnapView")

	if err := hotkey.Register(currentConfig.Hotkey, handleScreenshot); err != nil {
		log.Printf("Warning: Failed to register hotkey: %v", err)
		log.Println("The application will still run, but you'll need to manually configure the hotkey in your system settings.")
	}

	if currentConfig.AutoSave {
		if err := config.EnsureAutoSaveDir(); err != nil {
			log.Printf("Failed to create auto-save directory: %v", err)
		} else {
			capture.SetAutoSave(true, config.GetAutoSaveDir())
		}
	}

	go func() {
		for {
			select {
			case <-mViewPreview.ClickedCh:
				preview.OpenBrowser()
			case <-mCopyClipboard.ClickedCh:
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
			case <-mAutoSave.ClickedCh:
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

		if currentConfig.CopyToClipboard {
			go clipboard.CopyImage(imagePath)
		}
		preview.Show(imagePath)
	}()
}
