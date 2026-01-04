@echo off
echo Building SnapHook for Windows...
echo.

go build -ldflags "-H=windowsgui -s -w" -o snaphook.exe ./cmd/snaphook

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo Build successful!
    echo ========================================
    echo.
    echo Created: snaphook.exe
    echo.
    echo To run:
    echo   1. Double-click snaphook.exe
    echo   2. Look for the icon in your system tray
    echo   3. Press Ctrl+Shift+S to take a screenshot
    echo.
    echo To start on boot:
    echo   1. Press Win+R, type: shell:startup
    echo   2. Create a shortcut to snaphook.exe there
    echo.
    echo ========================================
) else (
    echo.
    echo Build failed! Make sure Go is installed.
    exit /b 1
)
