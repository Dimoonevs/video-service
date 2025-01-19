package utils

import "path/filepath"

func IsMP4(filename string) bool {
	return filepath.Ext(filename) == ".mp4"
}
