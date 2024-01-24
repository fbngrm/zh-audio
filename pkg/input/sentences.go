package input

import (
	"bufio"
	"context"
	"os"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
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
		if err := p.AudioDownloader.Fetch(context.Background(), sentence); err != nil {
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
		sentences = append(sentences, strings.TrimSpace(sentence))
	}
	return append(sentences, sentence), nil
}
