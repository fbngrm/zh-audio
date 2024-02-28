package input

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
	"github.com/fbngrm/zh-audio/pkg/google"
)

type DialogLine struct {
	Speaker string
	Text    string
}

type RawDialog struct {
	Speakers           map[string]struct{}
	Lines              []DialogLine
	Text               string // one line without speaker prefixes
	TextWithSpeaker    string
	TextWithOutSpeaker string
}

type DialogProcessor struct {
	AudioDownloader *audio.Downloader
}

func (p *DialogProcessor) GetAudio(path string) error {
	dialogs, err := p.loadDialogues(path)
	if err != nil {
		return err
	}
	for _, dialog := range dialogs {
		translation, err := google.Translate(dialog.Text)
		if err != nil {
			return err
		}

		dialogText := strings.ReplaceAll(dialog.Text, " 。", "。")
		if err := p.AudioDownloader.FetchEN(context.Background(), dialogText, translation); err != nil {
			return err
		}

		// generate sentences with different speakers
		voices := audio.GetVoicesZH(dialog.Speakers)
		var paths []string
		var slowPaths []string
		for _, line := range dialog.Lines {
			voice, ok := voices[line.Speaker]
			if !ok {
				fmt.Printf("could not find voice for speaker: %s\n", line.Speaker)
			}
			lineText := strings.ReplaceAll(line.Text, " 。", "。")
			path, err := p.AudioDownloader.FetchTmp(
				context.Background(),
				lineText,
				voice,
			)
			if err != nil {
				fmt.Println(err)
			}
			paths = append(paths, path)

			// generate slow audio with pause between words
			var wordPaths []string
			for _, word := range strings.Split(line.Text, " ") {
				path, err := p.AudioDownloader.FetchTmp(
					context.Background(),
					strings.ReplaceAll(word, " 。", "。"),
					voice,
				)
				if err != nil {
					fmt.Println(err)
				}
				wordPaths = append(wordPaths, path)
			}
			slowPath, err := p.AudioDownloader.JoinAndSaveSlowAudio(lineText, wordPaths)
			if err != nil {
				return err
			}
			slowPaths = append(slowPaths, slowPath)
		}
		if err := p.AudioDownloader.JoinAndSaveDialogAudio(dialogText, paths); err != nil {
			return err
		}
		if _, err := p.AudioDownloader.JoinAndSaveSlowAudio(dialogText, slowPaths); err != nil {
			return err
		}
	}
	return nil
}

func (p *DialogProcessor) loadDialogues(path string) ([]RawDialog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dialogs []RawDialog
	speakers := make(map[string]struct{})
	var lines []DialogLine
	var textWithSpeaker, textWithOutSpeaker, text string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rawLine := scanner.Text()
		if rawLine == "---" {
			dialogs = append(
				dialogs,
				RawDialog{
					Speakers:           speakers,
					Lines:              lines,
					TextWithSpeaker:    strings.TrimSpace(textWithSpeaker),
					TextWithOutSpeaker: strings.TrimSpace(textWithOutSpeaker),
					Text:               text,
				},
			)
			textWithSpeaker = ""
			textWithOutSpeaker = ""
			lines = []DialogLine{}
			speakers = make(map[string]struct{})
			text = ""
			continue
		}
		line := splitSpeakerAndText(rawLine)
		lines = append(lines, line)
		speakers[line.Speaker] = struct{}{}

		textWithSpeaker += rawLine
		textWithOutSpeaker += line.Text

		text += line.Text
		text += " "
	}
	dialogs = append(
		dialogs,
		RawDialog{
			Speakers:           speakers,
			Lines:              lines,
			TextWithSpeaker:    strings.TrimSpace(textWithSpeaker),
			TextWithOutSpeaker: strings.TrimSpace(textWithOutSpeaker),
			Text:               text,
		},
	)
	return dialogs, nil
}

func splitSpeakerAndText(line string) DialogLine {
	parts := []string{line}
	if strings.Contains(line, ":") {
		parts = strings.Split(line, ":")
	} else if strings.Contains(line, "：") {
		parts = strings.Split(line, "：")
	}
	if len(parts) == 1 {
		return DialogLine{
			"A",
			parts[0],
		}
	}
	return DialogLine{
		parts[0],
		parts[1],
	}
}
