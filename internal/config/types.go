package config

type Config struct {
	Hotkey          string `json:"hotkey"`
	AutoSave        bool   `json:"auto_save"`
	CopyToClipboard bool   `json:"copy_to_clipboard"`
	EnablePreview   bool   `json:"enable_preview"`
}
