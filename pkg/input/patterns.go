package input

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"golang.org/x/exp/slog"
)

type Grammar struct {
	SentenceBack    string    `json:"sentenceBack"`
	SentenceEnglish string    `json:"sentenceEnglish"`
	Pattern         string    `json:"pattern"`
	Note            string    `json:"note"`
	Structure       string    `json:"structure"`
	Examples        []Example `json:"examples"`
	Summary         []string  `json:"summary"`
}

type PatternProcessor struct {
	azureDownloader *audio.AzureClient
	concatenator    *audio.Concatenator
	cache           *audio.Cache
	outDir          string
}

func NewPatternProcessor(downloader *audio.AzureClient, concatenator *audio.Concatenator, cache *audio.Cache, outDir string) (*PatternProcessor, error) {
	out := filepath.Join(outDir, "patterns")
	if err := os.MkdirAll(out, os.ModePerm); err != nil {
		return nil, err
	}
	return &PatternProcessor{
		azureDownloader: downloader,
		concatenator:    concatenator,
		cache:           cache,
		outDir:          out,
	}, nil
}

func (p *PatternProcessor) replaceTextWithAudio(text, pause string) string {
	// Define a regex pattern to match sequences of Chinese characters
	chineseRe := regexp.MustCompile(`[\p{Han}]+`)
	// Update the regex pattern to match English phrases with surrounding punctuation included
	englishRe := regexp.MustCompile(`[A-Za-z]+(?:'[A-Za-z]+)?(?:[\s]*[,!?;:.]*[\s]*[A-Za-z]+(?:'[A-Za-z]+)?)*[,.!?;:]*`)

	// Replace English text with audio
	text = englishRe.ReplaceAllStringFunc(text, func(englishText string) string {
		audio := p.azureDownloader.PrepareEnglishQuery(englishText, pause)
		return audio
	})

	// Replace Chinese text with audio
	text = chineseRe.ReplaceAllStringFunc(text, func(chineseText string) string {
		audio := p.azureDownloader.PrepareQueryWithRandomVoice(chineseText, pause, false)
		return audio
	})

	return text
}

func (p *PatternProcessor) ConcatAudioFromCache(path string) error {
	patterns, err := loadFromDir(path)
	if err != nil {
		return err
	}

	for _, pa := range patterns {
		cachePath := p.cache.GetCachePath(pa.Pattern)
		tmpFile := audio.GetFilename(pa.Pattern)
		if !p.cache.IsInCache(cachePath) {
			query := cleanQuery(p.azureDownloader.PrepareQueryWithRandomVoice(pa.Pattern, "1500ms", true))
			slog.Debug("pattern not in cache, download with azure", "query", query)
			tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
			if err != nil {
				return err
			}
			p.concatenator.AddWithPause(tmpPath, 1500)
			p.concatenator.AddWithPause(tmpPath, 1500)
		} else {
			p.concatenator.AddWithPause(cachePath, 1500)
			p.concatenator.AddWithPause(cachePath, 1500)
		}

		note := removeDots(removeBracketsInclText(pa.Note))
		cachePath = p.cache.GetCachePath(note)
		tmpFile = audio.GetFilename(note)
		if !p.cache.IsInCache(cachePath) {
			query := cleanQuery(p.replaceTextWithAudio(note, "200ms"))
			slog.Debug("note not in cache, download with azure", "query", query)
			tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
			if err != nil {
				return err
			}
			p.concatenator.AddWithPause(tmpPath, 200)
		} else {
			p.concatenator.AddWithPause(cachePath, 200)
		}

		structure := removeAllQuotes(removeBracketsInclText(replaceSpecialChars(pa.Structure)))
		cachePath = p.cache.GetCachePath(structure)
		tmpFile = audio.GetFilename(structure)
		if !p.cache.IsInCache(cachePath) {
			query := cleanQuery(p.replaceTextWithAudio(structure, "500ms"))
			slog.Debug("structure not in cache, download with azure", "query", query)
			tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
			if err != nil {
				return err
			}
			p.concatenator.AddWithPause(tmpPath, 500)
		} else {
			p.concatenator.AddWithPause(cachePath, 500)
		}

		eng := "Here are a few examples"
		cachePath = p.cache.GetCachePath(eng)
		tmpFile = audio.GetFilename(eng)
		if !p.cache.IsInCache(cachePath) {
			query := p.azureDownloader.PrepareEnglishQuery(eng, "1000ms")
			slog.Debug("english not in cache, download with azure", "query", query)
			tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
			if err != nil {
				return err
			}
			p.concatenator.AddWithPause(tmpPath, 1000)
		} else {
			p.concatenator.AddWithPause(cachePath, 1000)
		}

		for _, e := range pa.Examples {
			cachePath := p.cache.GetCachePath(e.Chinese)
			tmpFile := audio.GetFilename(e.Chinese)
			if !p.cache.IsInCache(cachePath) {
				query := cleanQuery(p.azureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true))
				slog.Debug("example not in cache, download with azure", "query", query)
				tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
				if err != nil {
					return err
				}
				p.concatenator.AddWithPause(tmpPath, 2000)
				p.concatenator.AddWithPause(tmpPath, 2000)
			} else {
				p.concatenator.AddWithPause(cachePath, 2000)
				p.concatenator.AddWithPause(cachePath, 2000)
			}

			eng := removeWrappingSingleQuotes(e.English)
			cachePath = p.cache.GetCachePath(eng)
			tmpFile = audio.GetFilename(eng)
			if !p.cache.IsInCache(cachePath) {
				query := p.azureDownloader.PrepareEnglishQuery(eng, "2000ms")
				slog.Debug("english not in cache, download with azure", "query", query)
				tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
				if err != nil {
					return err
				}
				p.concatenator.AddWithPause(tmpPath, 2000)
			} else {
				p.concatenator.AddWithPause(cachePath, 2000)
			}

			cachePath = p.cache.GetCachePath(e.Chinese)
			tmpFile = audio.GetFilename(e.Chinese)
			if !p.cache.IsInCache(cachePath) {
				query := cleanQuery(p.azureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true))
				slog.Debug("example not in cache, download with azure", "query", query)
				tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
				if err != nil {
					return err
				}
				p.concatenator.AddWithPause(tmpPath, 2000)
			} else {
				p.concatenator.AddWithPause(cachePath, 2000)
			}
		}

		eng = "The most important points when using the pattern are:"
		cachePath = p.cache.GetCachePath(eng)
		tmpFile = audio.GetFilename(eng)
		if !p.cache.IsInCache(cachePath) {
			query := p.azureDownloader.PrepareEnglishQuery(eng, "200ms")
			slog.Debug("english not in cache, download with azure", "query", query)
			tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
			if err != nil {
				return err
			}
			p.concatenator.AddWithPause(tmpPath, 2000)
		} else {
			p.concatenator.AddWithPause(cachePath, 2000)
		}

		eng = strings.Join(pa.Summary, "\n")
		cachePath = p.cache.GetCachePath(eng)
		tmpFile = audio.GetFilename(eng)
		if !p.cache.IsInCache(cachePath) {
			query := p.azureDownloader.PrepareEnglishQuery(eng, "1500ms")
			slog.Debug("english not in cache, download with azure", "query", query)
			tmpPath, err := p.azureDownloader.Fetch(context.Background(), query, tmpFile)
			if err != nil {
				return err
			}
			p.concatenator.AddWithPause(tmpPath, 1500)
		} else {
			p.concatenator.AddWithPause(cachePath, 1500)
		}

		// add beep
		cachePath = p.cache.GetCachePath("peep")
		if !p.cache.IsInCache(cachePath) {
			slog.Error("missing peep file in cache", "path", cachePath)
		} else {
			p.concatenator.AddWithPause(cachePath, 1500)
		}

		if err := p.concatenator.Merge(filepath.Join(p.outDir, audio.GetFilename(pa.Pattern))); err != nil {
			slog.Error("concat files", "pattern", pa.Pattern, "error", err)
			continue
		}
		slog.Debug("concat files", "pattern", pa.Pattern)
	}
	return nil
}

func (p *PatternProcessor) GetAzureAudio(path string) error {
	patterns, err := loadFromDir(path)
	if err != nil {
		return err
	}
	for _, pa := range patterns {
		note := removeDots(removeBracketsInclText(pa.Note))
		query := p.azureDownloader.PrepareQueryWithRandomVoice(pa.Pattern, "1500ms", true)
		query += p.azureDownloader.PrepareQueryWithRandomVoice(pa.Pattern, "1500ms", true)
		query += p.replaceTextWithAudio(note, "200ms")
		query += p.replaceTextWithAudio(removeAllQuotes(removeBracketsInclText(replaceSpecialChars(pa.Structure))), "500ms")
		query += p.azureDownloader.PrepareEnglishQuery("Here are a few examples", "1000ms")
		for _, e := range pa.Examples {
			query += p.azureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
			query += p.azureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
			query += p.azureDownloader.PrepareEnglishQuery(removeWrappingSingleQuotes(e.English), "2000ms")
			query += p.azureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
		}
		query += p.azureDownloader.PrepareEnglishQuery("The most important points when using the pattern are", "200ms")
		query += p.azureDownloader.PrepareEnglishQuery(strings.Join(pa.Summary, "\n"), "1500ms")
		query = cleanQuery(query)
		fmt.Println(query)
		if _, err := p.azureDownloader.Fetch(context.Background(), query, audio.GetFilename(pa.Pattern)); err != nil {
			return err
		}
	}
	return nil
}

func loadFromDir(dir string) ([]Grammar, error) {
	var grammars []Grammar

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dir, file.Name())

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		var grammar Grammar
		if err := json.Unmarshal(data, &grammar); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON in file %s: %w", filePath, err)
		}

		grammars = append(grammars, grammar)
	}

	return grammars, nil
}

func replaceSpecialChars(text string) string {
	text = strings.ReplaceAll(text, "+", "followed by")
	text = strings.ReplaceAll(text, "[", "")
	text = strings.ReplaceAll(text, "]", "")

	// Define a regex pattern to match one or more whitespace characters
	re := regexp.MustCompile(`\s+`)
	// Replace all occurrences with a single space
	return re.ReplaceAllString(text, " ")
}

func removeBracketsInclText(text string) string {
	// Define a regex pattern to match text within brackets, including the brackets
	re := regexp.MustCompile(`\s*\(.*?\)`)
	// Replace all occurrences of the pattern with an empty string
	result := re.ReplaceAllString(text, "")
	return result
}

func removePunctuation(text string) string {
	// Define a regex pattern to match punctuation characters, except for those marking the end of a sentence
	// \p{P} matches any punctuation character in Unicode
	// We include commas in the removal list by excluding them along with `.,!?！？。` which mark sentence endings
	re := regexp.MustCompile(`[^\w\s\p{Han}]`)

	// Replace matched punctuation characters with an empty string
	result := re.ReplaceAllString(text, "")
	return result
}

// Function to remove all quotes (single and double) from a string
func removeAllQuotes(input string) string {
	// Define a regex pattern to match all single and double quotes
	re := regexp.MustCompile(`[\'\"]`)

	// Replace all matches with an empty string
	return re.ReplaceAllString(input, "")
}

func removeDots(input string) string {
	return strings.ReplaceAll(input, "…", "")
}
