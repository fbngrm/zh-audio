package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"github.com/fbngrm/zh-audio/pkg/input"
)

var audioDir = "./audio"
var inPath string
var t string

func main() {
	flag.StringVar(&inPath, "s", "", "source file")
	flag.StringVar(&t, "type", "s", "type of input [s=sentences|d=dialogs]")
	flag.Parse()

	if inPath == "" {
		fmt.Println("need input file, spcified with -s path/to/input")
		os.Exit(1)
	}

	audioDownloader, err := audio.NewAudioDownloader(audioDir)
	if err != nil {
		log.Panic(err)
	}

	if t == "dialogs" {
		dialogProcessor := input.DialogProcessor{
			AudioDownloader: audioDownloader,
		}
		if err := dialogProcessor.GetAudio(inPath); err != nil {
			log.Panic(err)
		}
		os.Exit(0)
	}

	sentenceProcessor := input.SentenceProcessor{
		AudioDownloader: audioDownloader,
	}
	if err := sentenceProcessor.GetAudio(inPath); err != nil {
		log.Panic(err)
	}
}
