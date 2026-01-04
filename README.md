# SnapHook

A lightweight, high-performance screenshot utility for Windows built with Go. SnapHook leverages concurrent goroutines to deliver instant screen captures with zero UI lag, running silently in your system tray.

## Overview

SnapHook is designed for users who need fast, reliable screenshot capture with automatic clipboard integration and intelligent multi-monitor support. Built using modern Go concurrency patterns, it provides a responsive experience even during rapid consecutive captures.

## Key Features

**Smart Multi-Monitor Capture**
Automatically detects which monitor your cursor is on and captures that display. Perfect for multi-monitor workflows.

**Instant Clipboard Integration**
Screenshots are automatically copied to your clipboard for immediate pasting. Toggle on/off from the system tray.

**Auto-Save to Pictures**
Optionally save all screenshots to `Pictures\SnapHook` with timestamped filenames for permanent storage.

**Live Browser Preview (Optional)**
Enable preview mode for super fast visibility of your screenshots! View captures instantly in a clean, dark-themed web interface with session history and one-click saving. Toggle on/off from the system tray.

**Configurable Hotkeys**
Default hotkey is Ctrl+Shift+S.

**System Tray Integration**
Minimal UI - runs silently in your system tray with right-click access to all settings.

**Auto-Start on Boot**
Optional Windows startup integration for always-available screenshots.

## Technical Stack

**Language:** Go
**Concurrency:** Goroutines for non-blocking screenshot processing and clipboard operations
**UI:** Native Windows API via syscall for system tray
**Capture:** Windows GDI through kbinani/screenshot library
**Preview:** Embedded HTTP server with Server-Sent Events for real-time updates
**Platform:** Windows 10+ (64-bit)

## Architecture Highlights

- **Concurrent Design** - Screenshot capture, clipboard copy, and browser preview run in parallel goroutines
- **Non-Blocking Operations** - Mutex-controlled screenshot flow allows rapid consecutive captures
- **Zero External Dependencies** - Single executable, no installation required

## Installation

Download `snaphook.exe` and run. No installation or dependencies required.

## Usage

1. Launch `snaphook.exe`
2. Look for the icon in your system tray
3. Press **Ctrl+Shift+S** to capture the monitor where your cursor is located
4. Screenshot is copied to clipboard instantly
5. Right-click system tray icon for settings:
   - **Enable Preview** - Turn on for super fast screenshot visibility in your browser
   - **Copy to Clipboard** - Automatically copy screenshots
   - **Auto-Save** - Save to Pictures\SnapHook
   - **Start on Boot** - Launch with Windows

## License

MIT License
