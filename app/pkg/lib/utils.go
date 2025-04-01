package lib

import (
	"flag"
	"path/filepath"
	"strings"
)

var (
	staticRootDir = flag.String("staticDir", "/var/www/file_service/", "static dir")
	publicHost    = flag.String("publicHost", "http://file.your-video-service.pp.ua/video/service/", "public host")
)

func IsMP4(filename string) bool {
	return filepath.Ext(filename) == ".mp4"
}

func GetVideoLocalLink(link string) string {
	return strings.ReplaceAll(link, *staticRootDir, *publicHost)
}

func GetVideoPublicLink(link string) string {
	return strings.ReplaceAll(link, *staticRootDir, *publicHost)
}
