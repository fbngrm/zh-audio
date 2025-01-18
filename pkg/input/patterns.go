package input

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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
	// Define a regex pattern to match sequences of Chinese characters along with punctuation
	chineseRe := regexp.MustCompile(`[\p{Han}][\p{Han}\p{P}]*`)
	// Define a regex pattern to match sequences of English words or contractions along with punctuation
	englishRe := regexp.MustCompile(`[A-Za-z]+(?:'[A-Za-z]+)?(?:\s+[A-Za-z]+(?:'[A-Za-z]+)?)?[\p{P}]*`)

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

func (p *PatternProcessor) splitByLanguageChange(input, pause string) string {
	// Define regex patterns for Chinese and non-Chinese text
	chineseRe := regexp.MustCompile(`[\p{Han}]+`)
	nonChineseRe := regexp.MustCompile(`[^\p{Han}]+`)

	// Initialize variables
	var chunks []string
	var currentChunk strings.Builder
	isChinese := false // Tracks if the current chunk is Chinese or not

	// Iterate through each character in the input
	for _, char := range input {
		charStr := string(char)

		if chineseRe.MatchString(charStr) {
			// If current character is Chinese
			if !isChinese && currentChunk.Len() > 0 {
				// Flush the non-Chinese chunk
				chunks = append(chunks,
					p.AzureDownloader.PrepareEnglishQuery(
						strings.TrimSpace(currentChunk.String()), pause))
				currentChunk.Reset()
			}
			isChinese = true
		} else if nonChineseRe.MatchString(charStr) {
			// If current character is non-Chinese
			if isChinese && currentChunk.Len() > 0 {
				// Flush the Chinese chunk
				chunks = append(chunks,
					p.AzureDownloader.PrepareQueryWithRandomVoice(
						strings.TrimSpace(currentChunk.String()), pause, false))
				currentChunk.Reset()
			}
			isChinese = false
		}
		// Append character to the current chunk
		currentChunk.WriteString(charStr)
	}

	if currentChunk.Len() == 0 {
		return strings.Join(chunks, "")
	}

	// Flush the last chunk if it exists
	if !isChinese {
		chunks = append(chunks,
			p.AzureDownloader.PrepareEnglishQuery(
				strings.TrimSpace(currentChunk.String()), pause))
	} else {
		chunks = append(chunks,
			p.AzureDownloader.PrepareQueryWithRandomVoice(
				strings.TrimSpace(currentChunk.String()), pause, false))
	}

	return strings.Join(chunks, "")
}

func (p *PatternProcessor) GetAzureAudio(path string) error {
	patterns, err := load(path)
	if err != nil {
		return err
	}
	for _, pa := range patterns {
		query := p.splitByLanguageChange(removeAllQuotes(removeBrackets(pa.Note)), "500ms")
		query += p.splitByLanguageChange(removeAllQuotes(removeBrackets(replaceSpecialChars(pa.Structure))), "500ms")
		query += p.AzureDownloader.PrepareEnglishQuery("Here are a few examples", "1000ms")
		for _, e := range pa.Examples {
			query += p.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "1500ms", true)
			query += p.AzureDownloader.PrepareEnglishQuery("which means", "300ms")
			query += p.AzureDownloader.PrepareEnglishQuery(e.English, "100ms")
			query += p.AzureDownloader.PrepareEnglishQuery("repeat the sentence", "300ms")
			query += p.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2500ms", true)
			query += p.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2500ms", true)
		}
		// query += p.splitByLanguageChange(removeAllQuotes(removeBrackets("Remember when using "+pa.Pattern)), "500ms")
		// query += p.splitByLanguageChange(removeAllQuotes(removeBrackets("the most important points are")), "500ms")
		// query += p.splitByLanguageChange(removeAllQuotes(removeBrackets(strings.Join(pa.Summary, "\n"))), "1500ms")
		fmt.Println(query)
		if err := p.AzureDownloader.Fetch(context.Background(), query, audio.GetFilename(pa.Pattern), true); err != nil {
			return err
		}
	}
	return nil
}

func load(filename string) ([]Grammar, error) {
	// Open the JSON file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var g []Grammar
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return g, nil
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
