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

type WordProcessor struct {
	AzureDownloader *audio.AzureClient
}

func (w *WordProcessor) GetAzureAudio(path string) error {
	words, err := loadWordsFromDir(path)
	if err != nil {
		return err
	}
	for _, wd := range words {
		if len(wd.HSK) == 0 && len(wd.Cedict) == 0 {
			return fmt.Errorf("word %s has no translation", wd.Chinese)
		}

		query := ""
		query += w.AzureDownloader.PrepareQueryWithRandomVoice(wd.Chinese, "1000ms", true)
		query += w.AzureDownloader.PrepareQueryWithRandomVoice(wd.Chinese, "1000ms", true)

		tones := ""
		for i, t := range wd.Tones {
			tones += t
			if i < len(wd.Tones)-1 {
				tones += ", followed by "
			}
		}
		if len(wd.Tones) == 1 {
			query += w.AzureDownloader.PrepareEnglishQuery("The tone is the "+tones, "1000ms")
		} else if len(wd.Tones) > 1 {
			query += w.AzureDownloader.PrepareEnglishQuery("The tones are "+tones, "1000ms")
		}
		query += w.AzureDownloader.PrepareQueryWithRandomVoice(wd.Chinese, "1000ms", true)

		wordEng := ""
		if len(wd.HSK) != 0 {
			for i, h := range wd.HSK {
				wordEng += h.HSKEnglish + " "
				if i < len(wd.HSK)-1 {
					wordEng += "or "
				}
			}
		}
		// Regex to match ", CL" followed by anything until the next whitespace
		re := regexp.MustCompile(`, CL[^\s]*`)
		if len(wd.HSK) == 0 && len(wd.Cedict) != 0 {
			for i, h := range wd.Cedict {
				wordEng += re.ReplaceAllString(h.CedictEnglish, "") + " "
				if i < len(wd.Cedict)-1 {
					wordEng += "or "
				}
			}
		}
		query += w.replaceTextWithAudio(wordEng, "1000ms")
		query += w.AzureDownloader.PrepareQueryWithRandomVoice(wd.Chinese, "1500ms", true)
		query += w.AzureDownloader.PrepareQueryWithRandomVoice(wd.Chinese, "1500ms", true)
		query += w.replaceTextWithAudio(removeWrappingSingleQuotes(wd.Note), "200ms")
		query += w.AzureDownloader.PrepareEnglishQuery("Here are a few example sentences", "1000ms")

		for _, e := range wd.Examples {
			query += w.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
			query += w.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
			query += w.AzureDownloader.PrepareEnglishQuery(removeWrappingSingleQuotes(e.English), "2000ms")
			query += w.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
		}

		query = cleanQuery(query)
		fmt.Println(strings.Count(query, "<voice"))
		// fmt.Println(query)
		if _, err := w.AzureDownloader.Fetch(context.Background(), query, audio.GetFilename(wd.Chinese)); err != nil {
			return err
		}
	}
	return nil
}

func (w *WordProcessor) replaceTextWithAudio(text, pause string) string {
	// Define a regex pattern to match sequences of Chinese characters
	chineseRe := regexp.MustCompile(`[\p{Han}]+`)
	// Update the regex pattern to match English phrases with surrounding punctuation included
	englishRe := regexp.MustCompile(`[A-Za-z]+(?:'[A-Za-z]+)?(?:[\s]*[,!?;:.]*[\s]*[A-Za-z]+(?:'[A-Za-z]+)?)*[,.!?;:]*`)

	// Replace English text with audio
	text = englishRe.ReplaceAllStringFunc(text, func(englishText string) string {
		audio := w.AzureDownloader.PrepareEnglishQuery(englishText, pause)
		return audio
	})

	// Replace Chinese text with audio
	text = chineseRe.ReplaceAllStringFunc(text, func(chineseText string) string {
		audio := w.AzureDownloader.PrepareQueryWithRandomVoice(chineseText, pause, false)
		return audio
	})

	return text
}

func loadWordsFromDir(dir string) ([]Word, error) {
	var words []Word

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

		var word Word
		if err := json.Unmarshal(data, &word); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON in file %s: %w", filePath, err)
		}

		words = append(words, word)
	}

	return words, nil
}

// Compile a regex to match everything between </voice> and the newline character
// Replace all matches of the regex with just the </voice> tag followed by a newline
// This effectively removes anything between </voice> and \n
func cleanQuery(query string) string {
	re := regexp.MustCompile(`</voice>.*\n`)
	return re.ReplaceAllString(query, "</voice>\n")
}
