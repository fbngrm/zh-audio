package audio

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"

	"github.com/faiface/beep/wav"
)

type Concatenator struct {
	Files  []string
	Pauses []int
}

func NewConcatenator() *Concatenator {
	return &Concatenator{
		Files:  make([]string, 0),
		Pauses: make([]int, 0),
	}
}

func (c *Concatenator) AddWithPause(file string, pause int) {
	c.Files = append(c.Files, file)
	c.Pauses = append(c.Pauses, pause)
}

func (c *Concatenator) Merge(outputFile string) error {
	if len(c.Files) == 0 {
		return fmt.Errorf("no input files provided")
	}

	if len(c.Pauses) != len(c.Files) {
		return fmt.Errorf("the number of pauses must match the number of files")
	}

	var streams []beep.Streamer
	var format beep.Format

	// Loop through each file and append to the streams slice
	for i, file := range c.Files {
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", file, err)
		}
		defer f.Close()

		stream, fFormat, err := mp3.Decode(f)
		if err != nil {
			return fmt.Errorf("failed to decode file %s: %v", file, err)
		}

		// Set format once from the first file
		if i == 0 {
			format = fFormat
		}

		streams = append(streams, stream)

		// Add pause after each file
		if i < len(c.Pauses) {
			pauseDuration := time.Duration(c.Pauses[i]) * time.Millisecond
			silence := beep.Silence(format.SampleRate.N(pauseDuration))
			streams = append(streams, silence)
		}
	}

	// Concatenate all streams
	finalStream := beep.Seq(streams...)

	// Create the output file
	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer out.Close()

	// Encode the final stream into WAV format
	err = wav.Encode(out, finalStream, format)
	if err != nil {
		return fmt.Errorf("failed to encode output file: %v", err)
	}

	return nil
}
