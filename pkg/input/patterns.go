package input

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
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
	AzureDownloader *audio.AzureClient
}

func (p *PatternProcessor) replaceTextWithAudio(text, pause string) string {
	// Define a regex pattern to match sequences of Chinese characters
	chineseRe := regexp.MustCompile(`[\p{Han}]+`)
	// Update the regex pattern to match English phrases with surrounding punctuation included
	englishRe := regexp.MustCompile(`[A-Za-z]+(?:'[A-Za-z]+)?(?:[\s]*[,!?;:.]*[\s]*[A-Za-z]+(?:'[A-Za-z]+)?)*[,.!?;:]*`)

	// Replace English text with audio
	text = englishRe.ReplaceAllStringFunc(text, func(englishText string) string {
		audio := p.AzureDownloader.PrepareEnglishQuery(englishText, pause)
		return audio
	})

	// Replace Chinese text with audio
	text = chineseRe.ReplaceAllStringFunc(text, func(chineseText string) string {
		audio := p.AzureDownloader.PrepareQueryWithRandomVoice(chineseText, pause, false)
		return audio
	})

	return text
}

func (p *PatternProcessor) GetAzureAudio(path string) error {
	patterns, err := loadFromDir(path)
	if err != nil {
		return err
	}
	for _, pa := range patterns {
		note := removeDots(removeBrackets(pa.Note))
		query := p.AzureDownloader.PrepareQueryWithRandomVoice(pa.Pattern, "1500ms", true)
		query += p.AzureDownloader.PrepareQueryWithRandomVoice(pa.Pattern, "1500ms", true)
		query += p.replaceTextWithAudio(note, "200ms")
		query += p.replaceTextWithAudio(removeAllQuotes(removeBrackets(replaceSpecialChars(pa.Structure))), "500ms")
		query += p.AzureDownloader.PrepareEnglishQuery("Here are a few examples", "1000ms")
		for _, e := range pa.Examples {
			query += p.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
			query += p.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
			query += p.AzureDownloader.PrepareEnglishQuery(removeWrappingSingleQuotes(e.English), "2000ms")
			query += p.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
		}
		query += p.AzureDownloader.PrepareEnglishQuery("The most important points when using the pattern are", "200ms")
		query += p.AzureDownloader.PrepareEnglishQuery(strings.Join(pa.Summary, "\n"), "1500ms")
		query = cleanQuery(query)
		fmt.Println(query)
		if err := p.AzureDownloader.Fetch(context.Background(), query, audio.GetFilename(pa.Pattern)); err != nil {
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
	result := strings.ReplaceAll(text, "+", "followed by")
	return result
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
