package sps

import (
	"os"

	"github.com/jasonkolodziej/go-castv2/sps/parse"
)

var DefaultConfPath = func(zoneName string) string {
	return "/etc/shairport-sync" + zoneName + ".conf"
}

var DefaultSystemDServicePath = func(zoneName string) string {
	return "/etc/shairport-sync" + zoneName + ".conf"
}

func OpenOriginalConfig() (f *os.File, size int64, err error) {
	return parse.LoadFile("/etc", "/shairport-sync.conf")
}
