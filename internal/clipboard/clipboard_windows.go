//go:build windows

package clipboard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
)

const (
	CF_DIB        = 8
	GMEM_MOVEABLE = 0x0002
)

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

func copyImage(imagePath string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode PNG: %w", err)
	}

	dibData, err := imageToDIB(img)
	if err != nil {
		return fmt.Errorf("failed to convert to DIB: %w", err)
	}

	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return fmt.Errorf("failed to open clipboard")
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	hMem, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, uintptr(len(dibData)))
	if hMem == 0 {
		return fmt.Errorf("failed to allocate memory")
	}

	pMem, _, _ := procGlobalLock.Call(hMem)
	if pMem == 0 {
		return fmt.Errorf("failed to lock memory")
	}

	dest := unsafe.Slice((*byte)(unsafe.Pointer(pMem)), len(dibData))
	copy(dest, dibData)
	procGlobalUnlock.Call(hMem)

	ret, _, _ = procSetClipboardData.Call(CF_DIB, hMem)
	if ret == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("failed to set clipboard data")
	}

	return nil
}

func imageToDIB(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rowSize := ((width*3 + 3) / 4) * 4
	imageSize := rowSize * height

	header := BITMAPINFOHEADER{
		Size:          40,
		Width:         int32(width),
		Height:        int32(height),
		Planes:        1,
		BitCount:      24,
		Compression:   0,
		SizeImage:     uint32(imageSize),
		XPelsPerMeter: 0,
		YPelsPerMeter: 0,
		ClrUsed:       0,
		ClrImportant:  0,
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, header)

	pixels := make([]byte, imageSize)
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			offset := (height-1-y)*rowSize + x*3
			pixels[offset] = byte(b >> 8)
			pixels[offset+1] = byte(g >> 8)
			pixels[offset+2] = byte(r >> 8)
		}
	}

	buf.Write(pixels)
	return buf.Bytes(), nil
}
