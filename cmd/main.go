package main

import (
	"flag"
	"log"
	"os"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"github.com/fbngrm/zh-audio/pkg/input"
)

var out = "./out"
var in string
var isDialog bool
var key string

func main() {
	flag.StringVar(&in, "src", "", "source file")
	flag.BoolVar(&isDialog, "d", false, "is this a dialog input")
	flag.Parse()

	if in == "" {
		log.Fatal("need input file, spcified with -src path/to/input")
	}

	audioDownloader, err := audio.NewAudioDownloader(out)
	if err != nil {
		log.Fatal(err)
	}

	if isDialog {
		dialogProcessor := input.DialogProcessor{
			AudioDownloader: audioDownloader,
		}
		if err := dialogProcessor.GetAudio(in); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	sentenceProcessor := input.SentenceProcessor{
		AudioDownloader: audioDownloader,
	}
	if err := sentenceProcessor.GetAudio(in); err != nil {
		log.Fatal(err)
	}
}
