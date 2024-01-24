package input

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fbngrm/zh-audio/pkg/audio"
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
		voices := audio.GetVoicesZH(dialog.Speakers)
		var paths []string
		for _, line := range dialog.Lines {
			voice, ok := voices[line.Speaker]
			if !ok {
				fmt.Printf("could not find voice for speaker: %s\n", line.Speaker)
			}
			path, err := p.AudioDownloader.FetchTmp(
				context.Background(),
				line.Text,
				voice,
			)
			if err != nil {
				fmt.Println(err)
			}
			paths = append(paths, path)
		}

		if err := p.AudioDownloader.JoinAndSaveDialogAudio(dialog.Text, paths); err != nil {
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
		textWithSpeaker += "<br>"
		textWithOutSpeaker += line.Text
		textWithOutSpeaker += "<br>"

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
	parts := []string{}
	if strings.Contains(line, ":") {
		parts = strings.Split(line, ":")
	} else if strings.Contains(line, "：") {
		parts = strings.Split(line, "：")
	}
	if len(parts) == 1 {
		return DialogLine{
			"",
			parts[0],
		}
	}
	return DialogLine{
		parts[0],
		parts[1],
	}
}
