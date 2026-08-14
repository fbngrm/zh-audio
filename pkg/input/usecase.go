package input

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"golang.org/x/exp/slog"
)

type UseCaseExample struct {
	Chinese string `json:"chinese"`
	English string `json:"english"`
}

type UseCaseProcessor struct {
	azureDownloader *audio.AzureClient
	segmenterDir    string
}

func NewUseCaseProcessor(downloader *audio.AzureClient, segmenterDir string) *UseCaseProcessor {
	return &UseCaseProcessor{
		azureDownloader: downloader,
		segmenterDir:    segmenterDir,
	}
}

// GenerateAudio reads use-case examples from a JSON file and, for each one,
// fetches three separate audio files from Azure TTS: the plain Chinese
// sentence, a segmented Chinese sentence (word-spaced, so Azure renders
// pauses between words), and the English translation. Unlike the pattern
// and sentence processors, these are not concatenated into one file - the
// mobile app streams each variant independently.
func (p *UseCaseProcessor) GenerateAudio(path string) error {
	examples, err := loadUseCases(path)
	if err != nil {
		return err
	}

	for _, ex := range examples {
		chinese := strings.ReplaceAll(ex.Chinese, " ", "")

		segmented, err := p.segment(chinese)
		if err != nil {
			return err
		}

		voice := p.azureDownloader.GetRandomVoice()

		regularQuery := p.azureDownloader.PrepareQuery(chinese, voice, "0ms", false)
		if _, err := p.azureDownloader.Fetch(context.Background(), regularQuery, audio.GetFilename(chinese)); err != nil {
			return err
		}

		segmentedFile := strings.TrimSuffix(audio.GetFilename(chinese), ".mp3") + "_segmented.mp3"
		segmentedQuery := p.azureDownloader.PrepareQueryRaw(segmented, voice, "0ms")
		if _, err := p.azureDownloader.Fetch(context.Background(), segmentedQuery, segmentedFile); err != nil {
			return err
		}

		englishQuery := p.azureDownloader.PrepareEnglishQuery(ex.English, "0ms")
		if _, err := p.azureDownloader.Fetch(context.Background(), englishQuery, audio.GetFilename(ex.English)); err != nil {
			return err
		}

		slog.Info("generated use-case audio", "chinese", chinese)
	}
	return nil
}

// segment runs the Chinese sentence through the Stanford PKU segmenter,
// producing whitespace-separated words. Azure's SSML renders whitespace as
// pauses, so this segmented text is what produces the "segmented" audio
// variant's word-by-word pacing.
func (p *UseCaseProcessor) segment(sentence string) (string, error) {
	tmpFile, err := os.CreateTemp("", "usecase-segment-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(sentence + "\n"); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	cmd := exec.Command(filepath.Join(p.segmenterDir, "segment.sh"), "pku", tmpFile.Name(), "UTF-8", "0")
	cmd.Dir = p.segmenterDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	segmented := strings.TrimSpace(string(out))
	segmented = strings.ReplaceAll(segmented, " 。", "")
	return segmented, nil
}

func loadUseCases(path string) ([]UseCaseExample, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var examples []UseCaseExample
	if err := json.Unmarshal(data, &examples); err != nil {
		return nil, err
	}
	return examples, nil
}
