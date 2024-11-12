package input

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
)

type PatternProcessor struct {
	AzureDownloader *audio.AzureClient
}

func (p *PatternProcessor) replaceTextWithAudio(text, pause string) string {
	// Define a regex pattern to match sequences of Chinese characters
	chineseRe := regexp.MustCompile(`[\p{Han}]+`)
	// Define a regex pattern to match sequences of English words or contractions
	englishRe := regexp.MustCompile(`[A-Za-z]+(?:'[A-Za-z]+)?(?:\s+[A-Za-z]+(?:'[A-Za-z]+)?)*`)
	// Replace English text with audio
	text = englishRe.ReplaceAllStringFunc(text, func(englishText string) string {
		audio := p.AzureDownloader.PrepareEnglishQuery(englishText, pause)
		return audio
	})

	// Replace Chinese text with audio
	text = chineseRe.ReplaceAllStringFunc(text, func(chineseText string) string {
		// audio := p.AzureDownloader.PrepareQuery(chineseText, "zh-CN-XiaochenNeural", pause, false)
		audio := p.AzureDownloader.PrepareQueryWithRandomVoice(chineseText, pause, false)
		return audio
	})

	return text
}

func (p *PatternProcessor) GetAzureAudio(path string) error {
	patterns, err := p.load(path)
	if err != nil {
		return err
	}
	for _, pa := range patterns {
		h := p.replaceTextWithAudio(removePunctuation(removeBrackets(pa.Head)), "300ms")
		u := p.replaceTextWithAudio(removePunctuation(removeBrackets(pa.Usage)), "1000ms")
		s := p.replaceTextWithAudio(removePunctuation(removeBrackets("The Syntax follows the pattern: ")), "500ms")
		s += p.replaceTextWithAudio(removePunctuation(removeBrackets(pa.Syntax)), "500ms")
		query := h + u
		query += s

		query += p.replaceTextWithAudio(removePunctuation(removeBrackets("Here are a few examples")), "1000ms")
		for _, e := range pa.Examples {
			query += p.replaceTextWithAudio(removePunctuation(removeBrackets(e.Ch)), "300ms")
			query += p.replaceTextWithAudio(removePunctuation(removeBrackets("which means")), "300ms")
			query += p.replaceTextWithAudio(removePunctuation(removeBrackets(e.En)), "1000ms")
			query += p.replaceTextWithAudio(removePunctuation(removeBrackets("repeat the sentence")), "0ms")
			query += p.replaceTextWithAudio(removePunctuation(removeBrackets(e.Ch)), "5000ms")
			query += p.replaceTextWithAudio(removePunctuation(removeBrackets(e.Ch)), "5000ms")
		}
		fmt.Println(query)
		if err := p.AzureDownloader.Fetch(context.Background(), query, audio.GetFilename(pa.Head), true); err != nil {
			return err
		}

	}
	return nil
}

func (p *PatternProcessor) load(filePath string) ([]Pattern, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file contents into a string
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return parsePatterns(string(content))
}

// Pattern represents the structure of the parsed pattern
type Pattern struct {
	Head     string
	Usage    string
	Syntax   string
	Examples []struct {
		Ch string
		En string
	}
}

func parsePatterns(input string) ([]Pattern, error) {
	var patterns []Pattern

	// Split input into raw patterns
	rawPatterns := strings.Split(input, "===")
	for _, rawPattern := range rawPatterns {
		rawPattern = strings.TrimSpace(rawPattern)
		if rawPattern == "" {
			continue
		}

		lines := strings.Split(rawPattern, "\n")
		if len(lines) < 5 {
			return nil, fmt.Errorf("unexpected pattern format")
		}

		var pattern Pattern
		pattern.Head = lines[0]
		pattern.Usage = lines[1]

		// Find Syntax which is the line right after the "--"
		syntaxIndex := findIndex(lines, "--") + 1
		if syntaxIndex < len(lines) {
			pattern.Syntax = lines[syntaxIndex]
		}

		// Find examples, they come after "---"
		examplesStart := findIndex(lines, "---") + 1
		for i := examplesStart; i < len(lines)-1; i += 2 {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				i--
				continue
			}
			// Assuming each example has the format: Chinese Text。English Text

			pattern.Examples = append(pattern.Examples, struct {
				Ch string
				En string
			}{
				Ch: strings.ReplaceAll(lines[i], " ", ""),
				En: strings.TrimSpace(lines[i+1]),
			})

		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// findIndex returns the index of the first occurrence of the target string in the lines
func findIndex(lines []string, target string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == target {
			return i
		}
	}
	return -1
}

func removeBrackets(text string) string {
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
