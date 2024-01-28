package input

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"github.com/fbngrm/zh-audio/pkg/deepl"
)

type SentenceProcessor struct {
	AudioDownloader *audio.Downloader
	Translator      *deepl.Client
}

func (p *SentenceProcessor) GetAudio(path string) error {
	sentences, err := p.loadSentences(path)
	if err != nil {
		return err
	}
	translations, err := p.Translator.Translate(sentences, 3)
	if err != nil {
		return err
	}
	if len(sentences) != len(translations) {
		return fmt.Errorf("translations mismatch %d:%d", len(sentences), len(translations))
	}
	for i, sentence := range sentences {
		if err := p.AudioDownloader.FetchEN(context.Background(), sentence, translations[i].Text); err != nil {
			return err
		}
		if err := p.AudioDownloader.FetchZH(context.Background(), sentence); err != nil {
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
