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
	AudioDownloader *audio.Downloader
}

func (p *SentenceProcessor) GetAudio(path string) error {
	sentences, err := p.loadSentences(path)
	if err != nil {
		return err
	}
	for _, sentence := range sentences {
		translation, err := google.Translate(sentence)
		if err != nil {
			return err
		}
		if err := p.AudioDownloader.FetchEN(context.Background(), sentence, translation); err != nil {
			return err
		}
		voice := audio.GetRandomVoiceZH()
		if err := p.AudioDownloader.FetchWithVoice(context.Background(), sentence, voice); err != nil {
			return err
		}

		// generate slow audio with pause between words
		var paths []string
		for _, word := range strings.Split(sentence, " ") {
			path, err := p.AudioDownloader.FetchTmp(
				context.Background(),
				word,
				voice,
			)
			if err != nil {
				fmt.Println(err)
			}
			paths = append(paths, path)
		}
		if _, err := p.AudioDownloader.JoinAndSaveSlowAudio(sentence, paths); err != nil {
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
		sentence = strings.ReplaceAll(sentence, " 。", "。")
		sentences = append(sentences, strings.TrimSpace(sentence))
	}
	return append(sentences, sentence), nil
}
