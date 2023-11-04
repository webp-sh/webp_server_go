package encoder

import (
	"path/filepath"
	"strings"

	"github.com/jeremytorres/rawparser"
)

func ConvertRawToJPG(rawPath, optimizedPath string) (string, bool) {
	if !strings.HasSuffix(strings.ToLower(rawPath), ".nef") {
		// Maybe can use rawParser to convert other raw files to jpg, but I haven't tested it
		return rawPath, false
	}
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
