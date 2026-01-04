//go:build windows

package hotkey

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	MOD_ALT      = 0x0001
	MOD_CONTROL  = 0x0002
	MOD_SHIFT    = 0x0004
	MOD_WIN      = 0x0008
	MOD_NOREPEAT = 0x4000

	WM_HOTKEY = 0x0312
	WM_QUIT   = 0x0012
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procRegisterHotKey     = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	procGetMessage         = user32.NewProc("GetMessageW")
	procPostThreadMessage  = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")

	hotkeyID     = 1
	isRunning    = false
	threadID     uint32
	loopExitChan chan struct{}
	hotkeyMutex  sync.Mutex
)

func parseHotkey(hotkey string) (uint32, uint32, error) {
	modifiers := uint32(MOD_NOREPEAT)
	var vkCode uint32

	switch hotkey {
	case "Ctrl+Shift+S":
		modifiers |= MOD_CONTROL | MOD_SHIFT
		vkCode = 0x53
	case "Ctrl+Alt+S":
		modifiers |= MOD_CONTROL | MOD_ALT
		vkCode = 0x53
	case "PrintScreen":
		vkCode = 0x2C
	default:
		return 0, 0, fmt.Errorf("unsupported hotkey: %s", hotkey)
	}

	return modifiers, vkCode, nil
}

func register(hotkey string) error {
	modifiers, vkCode, err := parseHotkey(hotkey)
	if err != nil {
		return err
	}

	log.Printf("Attempting to register hotkey: %s", hotkey)
	hotkeyMutex.Lock()
	isRunning = true
	loopExitChan = make(chan struct{})
	hotkeyMutex.Unlock()

	resultChan := make(chan error, 1)

	go func() {
		ret, _, err := procRegisterHotKey.Call(
			0,
			uintptr(hotkeyID),
			uintptr(modifiers),
			uintptr(vkCode),
		)

		if ret == 0 {
			log.Printf("Failed to register hotkey: %v", err)
			resultChan <- fmt.Errorf("failed to register hotkey: %v", err)
			return
		}

		log.Printf("Hotkey registered successfully: %s", hotkey)
		resultChan <- nil
		messageLoop()
		log.Println("Message loop has exited")
		close(loopExitChan)
	}()

	return <-resultChan
}

func messageLoop() {
	ret, _, _ := procGetCurrentThreadId.Call()
	hotkeyMutex.Lock()
	threadID = uint32(ret)
	hotkeyMutex.Unlock()
	log.Printf("Hotkey message loop started (threadID: %d)", threadID)

	msg := &MSG{}
	for {
		ret, _, _ := procGetMessage.Call(
			uintptr(unsafe.Pointer(msg)),
			0,
			0,
			0,
		)

		if ret == 0 || ret == uintptr(syscall.InvalidHandle) {
			log.Printf("Message loop exiting: ret=%d", ret)
			return
		}

		if msg.Message == WM_HOTKEY {
			log.Println("WM_HOTKEY received in message loop")
			if currentHandler != nil {
				go currentHandler()
			} else {
				log.Println("Warning: No handler set for hotkey")
			}
		} else if msg.Message == WM_QUIT {
			log.Println("WM_QUIT received, exiting message loop")
			return
		}
	}
}

func unregister() {
	hotkeyMutex.Lock()
	if !isRunning {
		hotkeyMutex.Unlock()
		return
	}

	isRunning = false
	tid := threadID
	exitChan := loopExitChan
	hotkeyMutex.Unlock()

	procUnregisterHotKey.Call(0, uintptr(hotkeyID))

	if tid != 0 {
		procPostThreadMessage.Call(uintptr(tid), WM_QUIT, 0, 0)
	}

	if exitChan != nil {
		select {
		case <-exitChan:
			log.Println("Hotkey message loop exited cleanly")
		case <-time.After(3 * time.Second):
			log.Println("Warning: Hotkey message loop did not exit within timeout")
		}
	}
}
