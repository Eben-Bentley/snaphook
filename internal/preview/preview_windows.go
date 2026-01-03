//go:build windows

package preview

func show(imagePath string) error {
	return ShowInBrowser(imagePath)
}
