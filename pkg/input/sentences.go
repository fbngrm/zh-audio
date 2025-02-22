package input

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"github.com/fbngrm/zh-audio/pkg/google"
)

type SentenceProcessor struct {
	gcpDownloader   *audio.GCPDownloader
	azureDownloader *audio.AzureClient
	concatenator    *audio.Concatenator
	cache           *audio.Cache
	outDir          string
}

func NewSentenceProcessor(
	downloader *audio.AzureClient,
	gcpDownloader *audio.GCPDownloader,
	concatenator *audio.Concatenator,
	cache *audio.Cache,
	outDir string) (*SentenceProcessor, error) {

	out := filepath.Join(outDir, "sentences")
	if err := os.MkdirAll(out, os.ModePerm); err != nil {
		return nil, err
	}
	return &SentenceProcessor{
		gcpDownloader:   gcpDownloader,
		azureDownloader: downloader,
		concatenator:    concatenator,
		cache:           cache,
		outDir:          out,
	}, nil
}

func (s *SentenceProcessor) ConcatAudioFromCache(path string) error {
	sentences, err := s.loadSentences(path)
	if err != nil {
		return err
	}
	for _, sentence := range sentences {
		translation, err := google.Translate(sentence)
		if err != nil {
			return err
		}

		cachePath := s.cache.GetCachePath(sentence)
		tmpFile := audio.GetFilename(sentence)
		if !s.cache.IsInCache(cachePath) {
			query := s.azureDownloader.PrepareQueryWithRandomVoice(sentence, "0ms", true)
			tmpPath, err := s.azureDownloader.Fetch(
				context.Background(),
				query,
				tmpFile)
			if err != nil {
				return err
			}
			s.concatenator.AddWithPause(tmpPath, 1500)
			s.concatenator.AddWithPause(tmpPath, 1500)
		} else {
			s.concatenator.AddWithPause(cachePath, 1500)
			s.concatenator.AddWithPause(cachePath, 1500)
		}

		cachePath = s.cache.GetCachePath(translation)
		tmpFile = audio.GetFilename(translation)
		if !s.cache.IsInCache(cachePath) {
			tmpPath, err := s.gcpDownloader.Fetch(context.Background(), tmpFile)
			if err != nil {
				return err
			}
			s.concatenator.AddWithPause(tmpPath, 1500)
		} else {
			s.concatenator.AddWithPause(cachePath, 1500)
		}

		cachePath = s.cache.GetCachePath(sentence)
		tmpFile = audio.GetFilename(sentence)
		if !s.cache.IsInCache(cachePath) {
			query := s.azureDownloader.PrepareQueryWithRandomVoice(sentence, "0ms", true)
			tmpPath, err := s.azureDownloader.Fetch(
				context.Background(),
				query,
				tmpFile)
			if err != nil {
				return err
			}
			s.concatenator.AddWithPause(tmpPath, 1500)
		} else {
			s.concatenator.AddWithPause(cachePath, 1500)
		}
	}
	return nil
}

func (p *SentenceProcessor) GetAzureAudio(path string) error {
	sentences, err := p.loadSentences(path)
	if err != nil {
		return err
	}
	for _, sentence := range sentences {
		translation, err := google.Translate(sentence)
		if err != nil {
			return err
		}
		if err := p.gcpDownloader.FetchEN(context.Background(), sentence, translation); err != nil {
			return err
		}

		// Use for single words
		// if err := p.GCPDownloader.FetchWithVoice(context.Background(), sentence, audio.VoicesZH[2]); err != nil {
		// 	return err
		// }

		query := p.azureDownloader.PrepareQueryWithRandomVoice(sentence, "0.0", true)
		if _, err := p.azureDownloader.Fetch(
			context.Background(),
			query,
			audio.GetFilename(sentence)); err != nil {
			return err
		}
	}
	return nil
}

func (p *SentenceProcessor) GetGCPAudio(path string) error {
	sentences, err := p.loadSentences(path)
	if err != nil {
		return err
	}
	for _, sentence := range sentences {
		translation, err := google.Translate(sentence)
		if err != nil {
			return err
		}
		if err := p.gcpDownloader.FetchEN(context.Background(), sentence, translation); err != nil {
			return err
		}
		voice := audio.GetRandomVoiceZH()
		if err := p.gcpDownloader.FetchWithVoice(context.Background(), sentence, voice); err != nil {
			return err
		}

		// generate slow audio with pause between words
		var paths []string
		for _, word := range strings.Split(sentence, " ") {
			path, err := p.gcpDownloader.FetchTmp(
				context.Background(),
				word,
				voice,
			)
			if err != nil {
				fmt.Println(err)
			}
			paths = append(paths, path)
		}
		if _, err := p.gcpDownloader.JoinAndSaveSlowAudio(sentence, paths); err != nil {
			return err
		}
	}
	return nil
}

func (p *SentenceProcessor) loadSentences(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var sentences []string
	var sentence string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		sentence = scanner.Text()
		if sentence == "" {
			continue
		}
		sentence = strings.ReplaceAll(sentence, " 。", "")
		sentences = append(sentences, strings.TrimSpace(sentence))
	}
	return append(sentences, sentence), nil
}
