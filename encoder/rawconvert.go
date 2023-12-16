package encoder

import (
	"path/filepath"

	"github.com/jeremytorres/rawparser"
)

func ConvertRawToJPG(rawPath, optimizedPath string) (string, bool) {
	parser, _ := rawparser.NewNefParser(true)
	info := &rawparser.RawFileInfo{
		File:    rawPath,
		Quality: 100,
		DestDir: optimizedPath,
	}
	_, err := parser.ProcessFile(info)
	if err == nil {
		_, file := filepath.Split(rawPath)
		return optimizedPath + file + "_extracted.jpg", true
	}
	return rawPath, false
}
