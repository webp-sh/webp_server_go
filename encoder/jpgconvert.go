package encoder

import (
	"github.com/jeremytorres/rawparser"
	"path/filepath"
	"strings"
)

func ConvertRawToJPG(rawPath, optimizedPath string) string {
	if !strings.HasSuffix(strings.ToLower(rawPath), ".nef") {
		// Maybe can use rawParser to convert other raw files to jpg, but I haven't tested it
		return rawPath
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
		return optimizedPath + file + "_extracted.jpg"
	}
	return rawPath
}
