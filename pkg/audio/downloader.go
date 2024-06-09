package audio

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

type GCPDownloader struct {
	dirEN   string
	dirZH   string
	dirSlow string
}

func NewGCPClient(dir string) (*GCPDownloader, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	dirZH := filepath.Join(dir, "zh")
	if err := os.MkdirAll(dirZH, os.ModePerm); err != nil {
		return nil, err
	}
	dirEN := filepath.Join(dir, "en")
	if err := os.MkdirAll(filepath.Join(dir, "en"), os.ModePerm); err != nil {
		return nil, err
	}
	dirSlow := filepath.Join(dir, "slow")
	if err := os.MkdirAll(dirSlow, os.ModePerm); err != nil {
		return nil, err
	}
	return &GCPDownloader{
		dirEN:   dirEN,
		dirZH:   dirZH,
		dirSlow: dirSlow,
	}, nil
}

func (p *GCPDownloader) GetFilename(query string) string {
	query = strings.ReplaceAll(query, " ", "")
	limit := math.Min(float64(len(query)), 150.0) // note: possible collisions
	return query[:int(limit)] + ".mp3"
}

func (p *GCPDownloader) GetOutpathZH(query string) string {
	return filepath.Join(p.dirZH, p.GetFilename(query))
}

func (p *GCPDownloader) GetOutpathEN(query string) string {
	return filepath.Join(p.dirEN, p.GetFilename(query))
}

func (p *GCPDownloader) getOutpathSlow(query string) string {
	return filepath.Join(p.dirSlow, p.GetFilename(query))
}

// download audio file from google text-to-speech api.
func (p *GCPDownloader) FetchZH(ctx context.Context, query string) error {
	return p.FetchWithVoice(ctx, query, GetRandomVoiceZH())
}

func (p *GCPDownloader) FetchEN(ctx context.Context, queryZH, query string) error {
	resp, err := fetch(ctx, query, GetRandomVoiceEN())
	if err != nil {
		return err
	}
	// the resp's AudioContent is binary
	err = ioutil.WriteFile(p.GetOutpathEN(queryZH), resp.AudioContent, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (p *GCPDownloader) FetchWithVoice(ctx context.Context, query string, voice *texttospeechpb.VoiceSelectionParams) error {
	resp, err := fetch(ctx, query, voice)
	if err != nil {
		return err
	}
	// the resp's AudioContent is binary
	err = ioutil.WriteFile(p.GetOutpathZH(query), resp.AudioContent, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (p *GCPDownloader) FetchTmp(ctx context.Context, query string, voice *texttospeechpb.VoiceSelectionParams,
) (string, error) {
	tmpFile, err := os.CreateTemp("", "zh")
	if err != nil {
		return "", fmt.Errorf("could not create tmp file: %v", err)
	}
	resp, err := fetch(ctx, query, voice)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(tmpFile.Name(), resp.AudioContent, os.ModePerm)
	if err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

func (p *GCPDownloader) JoinAndSaveSlowAudio(query string, inputPaths []string) (string, error) {
	outpath := p.getOutpathSlow(query)

	// ffmpeg command to join the MP3 files
	ffmpegArgs := []string{"-i", "concat:" + strings.Join(inputPaths, "|"), "-c", "copy", "-y", outpath}

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to join MP3 files: %v", err)
	}

	fmt.Printf("audio content written to file: %s\n", outpath)
	return outpath, nil
}

func (p *GCPDownloader) JoinAndSaveDialogAudio(query string, inputPaths []string) error {
	outpath := p.GetOutpathZH(query)

	// ffmpeg command to join the MP3 files
	ffmpegArgs := []string{"-i", "concat:" + strings.Join(inputPaths, "|"), "-c", "copy", "-y", outpath}

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to join MP3 files: %v", err)
	}

	fmt.Printf("audio content written to file: %s\n", outpath)
	return nil
}

func fetch(ctx context.Context, query string, voice *texttospeechpb.VoiceSelectionParams) (*texttospeechpb.SynthesizeSpeechResponse, error) {
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// perform the text-to-speech request on the text input with the selected
	// voice parameters and audio file type
	req := texttospeechpb.SynthesizeSpeechRequest{
		// set the text input to be synthesized
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: query},
		},
		// build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral")
		Voice: voice,
		// select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
			SpeakingRate:  0.80,
		},
	}
	return client.SynthesizeSpeech(ctx, &req)
}
