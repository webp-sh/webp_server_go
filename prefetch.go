package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func PrefetchImages(confImgPath string, ExhaustPath string, QUALITY string) {
	fmt.Println(`Prefetch will convert all your images to webp, it may take some time and consume a lot of CPU resource. Do you want to proceed(Y/n)`)
	reader := bufio.NewReader(os.Stdin)
	char, _, _ := reader.ReadRune() //y Y enter
	// maximum ongoing prefetch is depending on your core of CPU
	log.Printf("Prefetching using %d cores", jobs)
	var finishChan = make(chan int, jobs)
	for i := 0; i < jobs; i++ {
		finishChan <- 0
	}
	if char == 121 || char == 10 || char == 89 {
		//prefetch, recursive through the dir
		all := FileCount(confImgPath)
		count := 0
		err := filepath.Walk(confImgPath,
			func(picAbsPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// RawImagePath string, ImgFilename string, reqURI string
				proposedURI := strings.Replace(picAbsPath, confImgPath, "", 1)
				_, p2 := GenWebpAbs(picAbsPath, ExhaustPath, info.Name(), proposedURI)
				q, _ := strconv.ParseFloat(QUALITY, 32)
				_ = os.MkdirAll(path.Dir(p2), 0755)
				go WebpEncoder(picAbsPath, p2, float32(q), false, finishChan)
				count += <-finishChan
				//progress bar
				_, _ = fmt.Fprintf(os.Stdout, "[Webp Server started] - convert in progress: %d/%d\r", count, all)
				return nil
			})
		if err != nil {
			log.Println(err)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Prefetch completeY(^_^)Y\n\n")

}
