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
var isDialog, isSentences, isPatterns, isClozes, isWords, isUseCases bool
var key string
var ignoreChars = []string{"!", "！", "？", "?", "，", ",", ".", "。", "", " ", "、"}

func main() {
	flag.StringVar(&in, "src", "", "source file")
	flag.BoolVar(&isDialog, "d", false, "is this a dialog input")
	flag.BoolVar(&isPatterns, "p", false, "is this a pattern input")
	flag.BoolVar(&isSentences, "s", false, "is this a sentence input")
	flag.BoolVar(&isClozes, "c", false, "is this a cloze input")
	flag.BoolVar(&isWords, "w", false, "is this a words input")
	flag.BoolVar(&isUseCases, "u", false, "is this a use case input")
	flag.Parse()

	if in == "" {
		log.Fatal("need input file, spcified with -src path/to/input")
	}

	audioCacheDir := os.Getenv("AUDIO_CACHE_DIR")
	if audioCacheDir == "" {
		log.Fatal("Environment variable AUDIO_CACHE_DIR is not set")
	}
	ttsClient, err := audio.NewTTSClient(out, ignoreChars)
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
			GCPDownloader: gcpClient,
			TTSDownloader: ttsClient,
		}
		if err := dialogProcessor.GetAzureAudio(in); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	if isSentences {
		sentenceProcessor, err := input.NewSentenceProcessor(
			ttsClient,
			gcpClient,
			concatenator,
			cache,
			out,
		)
		if err != nil {
			log.Fatal(err)
		}
		// if err := sentenceProcessor.GetAzureAudio(in); err != nil {
		// 	log.Fatal(err)
		// }
		if err := sentenceProcessor.ConcatAudioFromCache(in); err != nil {
			log.Fatal(err)
		}
	}
	if isPatterns {
		patternProcessor, err := input.NewPatternProcessor(
			ttsClient,
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
			TTSDownloader: ttsClient,
		}
		if err := clozesProcessor.GetAzureAudio(in); err != nil {
			log.Fatal(err)
		}
	}
	if isWords {
		wordsProcessor := input.WordProcessor{
			TTSDownloader: ttsClient,
		}
		if err := wordsProcessor.GetAzureAudio(in); err != nil {
			log.Fatal(err)
		}
	}
	if isUseCases {
		useCaseProcessor := input.NewUseCaseProcessor(ttsClient, "../stanford-segmenter")
		if err := useCaseProcessor.GenerateAudio(in); err != nil {
			log.Fatal(err)
		}
	}
}
