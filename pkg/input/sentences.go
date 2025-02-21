package input

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"github.com/fbngrm/zh-audio/pkg/google"
)

type SentenceProcessor struct {
	GCPDownloader   *audio.GCPDownloader
	AzureDownloader *audio.AzureClient
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
		if err := p.GCPDownloader.FetchEN(context.Background(), sentence, translation); err != nil {
			return err
		}

		// Use for single words
		// if err := p.GCPDownloader.FetchWithVoice(context.Background(), sentence, audio.VoicesZH[2]); err != nil {
		// 	return err
		// }

		query := p.AzureDownloader.PrepareQueryWithRandomVoice(sentence, "0.0", true)
		if _, err := p.AzureDownloader.Fetch(
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
		if err := p.GCPDownloader.FetchEN(context.Background(), sentence, translation); err != nil {
			return err
		}
		voice := audio.GetRandomVoiceZH()
		if err := p.GCPDownloader.FetchWithVoice(context.Background(), sentence, voice); err != nil {
			return err
		}

		// generate slow audio with pause between words
		var paths []string
		for _, word := range strings.Split(sentence, " ") {
			path, err := p.GCPDownloader.FetchTmp(
				context.Background(),
				word,
				voice,
			)
			if err != nil {
				fmt.Println(err)
			}
			paths = append(paths, path)
		}
		if _, err := p.GCPDownloader.JoinAndSaveSlowAudio(sentence, paths); err != nil {
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
