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
var ignoreChars = []string{"!", "！", "？", "?", "，", ",", ".", "。", "", " ", "、"}

func main() {
	flag.StringVar(&in, "src", "", "source file")
	flag.BoolVar(&isDialog, "d", false, "is this a dialog input")
	flag.Parse()

	if in == "" {
		log.Fatal("need input file, spcified with -src path/to/input")
	}

	azureApiKey := os.Getenv("SPEECH_KEY")
	if azureApiKey == "" {
		log.Fatal("Environment variable SPEECH_KEY is not set")
	}
	azureClient, err := audio.NewAzureClient(azureApiKey, out, ignoreChars)
	if err != nil {
		log.Fatal(err)
	}

	gcpClient, err := audio.NewGCPClient(out)
	if err != nil {
		log.Fatal(err)
	}

	if isDialog {
		dialogProcessor := input.DialogProcessor{
			GCPDownloader:   gcpClient,
			AzureDownloader: azureClient,
		}
		// if err := dialogProcessor.GetGCPAudio(in); err != nil {
		// 	log.Fatal(err)
		// }
		if err := dialogProcessor.GetAzureAudio(in); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	sentenceProcessor := input.SentenceProcessor{
		GCPDownloader:   gcpClient,
		AzureDownloader: azureClient,
	}
	if err := sentenceProcessor.GetAzureAudio(in); err != nil {
		log.Fatal(err)
	}
	// if err := sentenceProcessor.GetGCPAudio(in); err != nil {
	// 	log.Fatal(err)
	// }
}
