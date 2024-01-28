package main

import (
	"flag"
	"log"
	"os"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"github.com/fbngrm/zh-audio/pkg/deepl"
	"github.com/fbngrm/zh-audio/pkg/input"
)

var out = "./out"
var in string
var isDialog bool
var key string

func main() {
	apiKey := os.Getenv("DEEPL_API_KEY")
	if apiKey == "" {
		log.Fatal("Environment variable DEEPL_API_KEY is not set")
	}
	flag.StringVar(&in, "src", "", "source file")
	flag.BoolVar(&isDialog, "d", false, "is this a dialog input")
	flag.Parse()

	if in == "" {
		log.Fatal("need input file, spcified with -s path/to/input")
	}

	audioDownloader, err := audio.NewAudioDownloader(out)
	if err != nil {
		log.Fatal(err)
	}

	translator := deepl.NewClient(apiKey)

	if isDialog {
		dialogProcessor := input.DialogProcessor{
			AudioDownloader: audioDownloader,
			Translator:      translator,
		}
		if err := dialogProcessor.GetAudio(in); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	sentenceProcessor := input.SentenceProcessor{
		AudioDownloader: audioDownloader,
		Translator:      translator,
	}
	if err := sentenceProcessor.GetAudio(in); err != nil {
		log.Fatal(err)
	}
}
