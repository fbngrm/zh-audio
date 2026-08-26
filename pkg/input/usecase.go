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
	// ID is optional. When set (and destDir is configured), output for this
	// example is written directly to destDir as {id}.mp3/{id}_segmented.mp3
	// instead of zh-audio's own text-keyed staging dir - so a consumer like
	// phrase-app doesn't need a separate import/copy pass, and zh-audio's
	// out/ dir doesn't pile up with a copy of every consumer's audio.
	ID      string `json:"id,omitempty"`
	Chinese string `json:"chinese"`
	English string `json:"english"`
}

type UseCaseProcessor struct {
	ttsDownloader *audio.TTSClient
	segmenterDir  string
	destDir       string
}

func NewUseCaseProcessor(downloader *audio.TTSClient, segmenterDir string, destDir string) *UseCaseProcessor {
	return &UseCaseProcessor{
		ttsDownloader: downloader,
		segmenterDir:  segmenterDir,
		destDir:       destDir,
	}
}

// GenerateAudio reads use-case examples from a JSON file and, for each one,
// fetches audio from Azure TTS for the plain Chinese sentence and a
// segmented Chinese sentence (word-spaced, so Azure renders pauses between
// words); an English narration file is also generated for text-keyed
// examples (id-less), matching the original zh-patterns/mobile use case.
// Unlike the pattern and sentence processors, these are not concatenated
// into one file - the mobile app streams each variant independently.
//
// For id-keyed examples (destDir set and ex.ID non-empty), each file is
// moved straight to destDir as {id}.mp3/{id}_segmented.mp3 as soon as it's
// generated, and any file already present there is skipped rather than
// regenerated - so an interrupted run can simply be re-run to resume. No
// English narration is generated in this mode since phrase-app only wants
// the Chinese normal/segmented pair.
func (p *UseCaseProcessor) GenerateAudio(path string) error {
	examples, err := loadUseCases(path)
	if err != nil {
		return err
	}

	for _, ex := range examples {
		chinese := strings.ReplaceAll(ex.Chinese, " ", "")
		useDest := p.destDir != "" && ex.ID != ""

		normalDest := ex.ID + ".mp3"
		segmentedDest := ex.ID + "_segmented.mp3"
		if useDest && p.existsInDest(normalDest) && p.existsInDest(segmentedDest) {
			slog.Info("skipping, already generated", "id", ex.ID, "chinese", chinese)
			continue
		}

		segmented, err := p.segment(chinese)
		if err != nil {
			return err
		}

		// phrase-app's audio wants a fixed, consistent narrator rather than
		// a voice picked at random per sentence.
		voice := "zh-CN-XiaoxiaoNeural"

		if !useDest || !p.existsInDest(normalDest) {
			regularQuery := p.ttsDownloader.PrepareQuery(chinese, voice, "0ms", false)
			outPath, err := p.ttsDownloader.Fetch(context.Background(), regularQuery, audio.GetFilename(chinese))
			if err != nil {
				return err
			}
			if useDest {
				if err := p.moveToDest(outPath, normalDest); err != nil {
					return err
				}
			}
		}

		if !useDest || !p.existsInDest(segmentedDest) {
			segmentedFile := strings.TrimSuffix(audio.GetFilename(chinese), ".mp3") + "_segmented.mp3"
			segmentedQuery := p.ttsDownloader.PrepareQueryRaw(segmented, voice, "0ms")
			outPath, err := p.ttsDownloader.Fetch(context.Background(), segmentedQuery, segmentedFile)
			if err != nil {
				return err
			}
			if useDest {
				if err := p.moveToDest(outPath, segmentedDest); err != nil {
					return err
				}
			}
		}

		if !useDest {
			englishQuery := p.ttsDownloader.PrepareEnglishQuery(ex.English, "0ms")
			if _, err := p.ttsDownloader.Fetch(context.Background(), englishQuery, audio.GetFilename(ex.English)); err != nil {
				return err
			}
		}

		slog.Info("generated use-case audio", "chinese", chinese)
	}
	return nil
}

func (p *UseCaseProcessor) existsInDest(filename string) bool {
	info, err := os.Stat(filepath.Join(p.destDir, filename))
	return err == nil && info.Size() > 0
}

func (p *UseCaseProcessor) moveToDest(srcPath, filename string) error {
	if err := os.MkdirAll(p.destDir, os.ModePerm); err != nil {
		return err
	}
	return os.Rename(srcPath, filepath.Join(p.destDir, filename))
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
