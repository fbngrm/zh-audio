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
var isDialog, isSentences, isPatterns, isClozes, isWords bool
var key string
var ignoreChars = []string{"!", "！", "？", "?", "，", ",", ".", "。", "", " ", "、"}

func main() {
	flag.StringVar(&in, "src", "", "source file")
	flag.BoolVar(&isDialog, "d", false, "is this a dialog input")
	flag.BoolVar(&isPatterns, "p", false, "is this a pattern input")
	flag.BoolVar(&isSentences, "s", false, "is this a sentence input")
	flag.BoolVar(&isClozes, "c", false, "is this a cloze input")
	flag.BoolVar(&isWords, "w", false, "is this a words input")
	flag.Parse()

	if in == "" {
		log.Fatal("need input file, spcified with -src path/to/input")
	}

	audioCacheDir := os.Getenv("AUDIO_CACHE_DIR")
	if audioCacheDir == "" {
		log.Fatal("Environment variable AUDIO_CACHE_DIR is not set")
	}
	azureApiKey := os.Getenv("SPEECH_KEY")
	if azureApiKey == "" {
		log.Fatal("Environment variable SPEECH_KEY is not set")
	}
	azureEndpoint := os.Getenv("AZURE_ENDPOINT")
	if azureEndpoint == "" {
		log.Fatal("Environment variable AZURE_ENDPOINT is not set")
	}
	azureClient, err := audio.NewAzureClient(azureApiKey, azureEndpoint, out, ignoreChars)
	if err != nil {
		log.Fatal(err)
	}

	gcpClient, err := audio.NewGCPClient(out)
	if err != nil {
		log.Fatal(err)
	}

	concatenator := audio.NewConcatenator()

	cache := &audio.Cache{
		AudioCacheDir: audioCacheDir,
	}

	if isDialog {
		dialogProcessor := input.DialogProcessor{
			GCPDownloader:   gcpClient,
			AzureDownloader: azureClient,
		}
		if err := dialogProcessor.GetAzureAudio(in); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	if isSentences {
		sentenceProcessor := input.SentenceProcessor{
			GCPDownloader:   gcpClient,
			AzureDownloader: azureClient,
		}
		if err := sentenceProcessor.GetAzureAudio(in); err != nil {
			log.Fatal(err)
		}
	}
	if isPatterns {
		patternProcessor, err := input.NewPatternProcessor(
			azureClient,
			concatenator,
			cache,
			out,
		)
		if err != nil {
			log.Fatal(err)
		}
		// if err := patternProcessor.GetAzureAudio(in); err != nil {
		// 	log.Fatal(err)
		// }
		if err := patternProcessor.ConcatAudioFromCache(in); err != nil {
			log.Fatal(err)
		}
	}
	if isClozes {
		clozesProcessor := input.ClozeProcessor{
			AzureDownloader: azureClient,
		}
		if err := clozesProcessor.GetAzureAudio(in); err != nil {
			log.Fatal(err)
		}
	}
	if isWords {
		wordsProcessor := input.WordProcessor{
			AzureDownloader: azureClient,
		}
		if err := wordsProcessor.GetAzureAudio(in); err != nil {
			log.Fatal(err)
		}
	}
}
