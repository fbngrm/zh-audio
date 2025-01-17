package input

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/fbngrm/zh-audio/pkg/audio"
)

type CedictEntry struct {
	CedictEnglish string `json:"cedict_en"`
}

type HSKEntry struct {
	HSKEnglish string `json:"hsk_en"`
}

type Example struct {
	Chinese string `json:"chinese"`
	English string `json:"hsk_en"`
}

type Word struct {
	Chinese     string        `json:"chinese"`
	English     string        `json:"english"`
	Cedict      []CedictEntry `json:"cedict"`
	HSK         []HSKEntry    `json:"hsk"`
	Note        string        `json:"note"`
	Translation string        `json:"translation"` // this is coming from data/translations file
	Examples    []Example     `json:"examples"`
	Tones       []string      `json:"tones"`
}

type Cloze struct {
	SentenceBack string `json:"chinese"`
	English      string `json:"english"`
	Grammar      string `json:"grammar"`
	Note         string `json:"note"`
	Word         Word   `json:"word"`
}

type ClozeProcessor struct {
	AzureDownloader *audio.AzureClient
}

func (c *ClozeProcessor) GetAzureAudio(path string) error {
	clozes, err := loadClozesFromFile(path)
	if err != nil {
		return err
	}
	for _, cl := range clozes {
		if len(cl.Word.HSK) == 0 && len(cl.Word.Cedict) == 0 {
			return fmt.Errorf("word %s has no translation", cl.Word.Chinese)
		}

		query := ""
		query += c.AzureDownloader.PrepareQueryWithRandomVoice(cl.Word.Chinese, "2000ms", false)
		query += c.AzureDownloader.PrepareQueryWithRandomVoice(cl.Word.Chinese, "1000ms", false)

		tones := ""
		for i, t := range cl.Word.Tones {
			tones += t
			if i < len(cl.Word.Tones)-1 {
				tones += ", followed by "
			}
		}
		if len(cl.Word.Tones) == 1 {
			query += w.AzureDownloader.PrepareEnglishQuery("The tone is the "+tones, "1000ms")
		} else if len(cl.Word.Tones) > 1 {
			query += w.AzureDownloader.PrepareEnglishQuery("The tones are "+tones, "1000ms")
		}
		query += w.AzureDownloader.PrepareQueryWithRandomVoice(cl.Word.Chinese, "2000ms", true)

		wordEng := ""
		if len(cl.Word.HSK) != 0 {
			for i, h := range cl.Word.HSK {
				wordEng += h.HSKEnglish + " "
				if i < len(cl.Word.HSK)-1 {
					wordEng += "or "
				}
			}
		}
		if len(cl.Word.HSK) == 0 && len(cl.Word.Cedict) != 0 {
			for i, h := range cl.Word.Cedict {
				wordEng += h.CedictEnglish + " "
				if i < len(cl.Word.Cedict)-1 {
					wordEng += "or "
				}
			}
		}
		query += w.replaceTextWithAudio(wordEng, "1000ms")
		query += w.AzureDownloader.PrepareQueryWithRandomVoice(cl.Word.Chinese, "1500ms", true)
		query += w.AzureDownloader.PrepareQueryWithRandomVoice(cl.Word.Chinese, "1500ms", true)
		query += c.replaceTextWithAudio(removeWrappingSingleQuotes(cl.Word.Note), "200ms")
		query += w.AzureDownloader.PrepareEnglishQuery("Here are a few example sentences", "1000ms")

		query += c.AzureDownloader.PrepareQueryWithRandomVoice(cl.SentenceBack, "2000ms", true)
		query += c.AzureDownloader.PrepareQueryWithRandomVoice(cl.SentenceBack, "2000ms", true)
		query += c.AzureDownloader.PrepareEnglishQuery(cl.English, "2000ms")
		query += c.AzureDownloader.PrepareQueryWithRandomVoice(cl.SentenceBack, "2000ms", true)

		for _, e := range cl.Word.Examples {
			query += c.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
			query += c.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
			query += c.AzureDownloader.PrepareEnglishQuery(e.English, "2000ms")
			query += c.AzureDownloader.PrepareQueryWithRandomVoice(e.Chinese, "2000ms", true)
		}

		// fmt.Println(query)
		if err := c.AzureDownloader.Fetch(context.Background(), query, audio.GetFilename(cl.SentenceBack), true); err != nil {
			return err
		}
	}
	return nil
}

func (c *ClozeProcessor) replaceTextWithAudio(text, pause string) string {
	// Define a regex pattern to match sequences of Chinese characters
	chineseRe := regexp.MustCompile(`[\p{Han}]+`)
	// Update the regex pattern to match English phrases with surrounding punctuation included
	englishRe := regexp.MustCompile(`[A-Za-z]+(?:'[A-Za-z]+)?(?:[\s]*[,!?;:.]*[\s]*[A-Za-z]+(?:'[A-Za-z]+)?)*[,.!?;:]*`)

	// Replace English text with audio
	text = englishRe.ReplaceAllStringFunc(text, func(englishText string) string {
		audio := c.AzureDownloader.PrepareEnglishQuery(englishText, pause)
		return audio
	})

	// Replace Chinese text with audio
	text = chineseRe.ReplaceAllStringFunc(text, func(chineseText string) string {
		audio := c.AzureDownloader.PrepareQueryWithRandomVoice(chineseText, pause, false)
		return audio
	})

	return text
}

func loadClozesFromFile(filename string) ([]Cloze, error) {
	// Open the JSON file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file content
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal JSON into a slice of Cloze structs
	var clozes []Cloze
	if err := json.Unmarshal(data, &clozes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return clozes, nil
}

func main() {
	filename := "clozes.json"
	clozes, err := loadClozesFromFile(filename)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print the loaded clozes
	for i, cloze := range clozes {
		fmt.Printf("Cloze #%d: %+v\n", i+1, cloze)
	}
}

func removeWrappingSingleQuotes(input string) string {
	// Define a regex pattern to match single quotes wrapping around Chinese characters
	re := regexp.MustCompile(`'([\p{Han}]+)'`)

	// Replace matches with just the Chinese characters, removing the wrapping single quotes
	return re.ReplaceAllString(input, "$1")
}
