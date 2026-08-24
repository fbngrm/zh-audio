package audio

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

// Chinese speech is slowed to 70% of normal speed for language-learning
// clarity (matches the previous Azure prosody rate="0.7"); English narration
// plays at normal speed, also matching the previous Azure client.
const (
	zhRatePercent = "-30%"
	enRatePercent = "+0%"
)

// segFieldDelim/segDelim separate a Fetch query into (voice, rate, pause,
// text) segments. Control characters are used since they can't appear in the
// corpus text (JSON strings, translations) that flows through here.
const (
	segFieldDelim = "\x02"
	segDelim      = "\x01"
)

// TTSClient generates Chinese/English speech locally via the edge-tts CLI
// (https://github.com/rany2/edge-tts), which streams the same Azure neural
// voices Cognitive Services uses, for free - no API key required.
//
// A Fetch query is not SSML - it's the small segment encoding built by
// encodeSegment/the Prepare* methods below (one voice+rate+pause+text unit
// per synthesis call). This lets a single Fetch cover several voices/pauses
// (e.g. a multi-speaker dialog line, or word pronunciation bookended by
// English narration) the way one multi-<voice> Azure SSML request used to:
// Fetch synthesizes each segment separately via edge-tts, then concatenates
// them (with each segment's pause inserted after it) into one output file.
type TTSClient struct {
	AudioDir    string
	ignoreChars []string
}

func NewTTSClient(dir string, ignoreChars []string) (*TTSClient, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "zh")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}
	return &TTSClient{
		AudioDir:    dir,
		ignoreChars: ignoreChars,
	}, nil
}

// we support 4 different voices only. zh-CN-XiaochenNeural and
// zh-CN-YunyiMultilingualNeural (the previous Azure client's third and
// fourth voices) aren't served by edge-tts's underlying Edge read-aloud
// service (confirmed via `edge-tts --list-voices`), so they're swapped for
// zh-CN-XiaoyiNeural and zh-CN-YunyangNeural, which are.
var Voices = []string{
	"zh-CN-XiaoxiaoNeural", // female
	"zh-CN-YunjianNeural",  // male
	"zh-CN-XiaoyiNeural",   // female
	"zh-CN-YunyangNeural",  // male
}

func (c *TTSClient) GetRandomVoice() string {
	rand.Seed(time.Now().UnixNano()) // initialize global pseudo random generator
	return Voices[rand.Intn(len(Voices))]
}

func (c *TTSClient) GetVoices(speakers map[string]struct{}) map[string]string {
	v := make(map[string]string)
	var i int
	for speaker := range speakers {
		v[speaker] = Voices[i]
		i++
	}
	return v
}

// Fetch synthesizes the segments encoded in query (see encodeSegment) and
// writes the merged result to AudioDir/filename.
func (c *TTSClient) Fetch(ctx context.Context, query, filename string) (string, error) {
	if contains(c.ignoreChars, query) {
		return "", nil
	}
	if err := os.MkdirAll(c.AudioDir, os.ModePerm); err != nil {
		return "", err
	}
	outPath := filepath.Join(c.AudioDir, filename)

	segments := decodeSegments(query)
	if len(segments) == 0 {
		return "", fmt.Errorf("no tts segments in query")
	}

	concat := NewConcatenator()
	var tmpFiles []string
	defer func() {
		for _, f := range tmpFiles {
			os.Remove(f)
		}
	}()

	for _, seg := range segments {
		tmp, err := c.synthesize(ctx, seg.voice, seg.ratePercent, seg.text)
		if err != nil {
			return "", err
		}
		tmpFiles = append(tmpFiles, tmp)
		concat.AddWithPause(tmp, seg.pauseMs)
	}

	tmpWav, err := os.CreateTemp("", "tts-merged-*.wav")
	if err != nil {
		return "", err
	}
	tmpWav.Close()
	defer os.Remove(tmpWav.Name())

	if err := concat.Merge(tmpWav.Name()); err != nil {
		return "", fmt.Errorf("merging tts segments: %w", err)
	}
	if err := transcodeToMP3(tmpWav.Name(), outPath); err != nil {
		return "", fmt.Errorf("transcoding merged audio: %w", err)
	}

	slog.Info("audio content generated", "path", outPath)
	return outPath, nil
}

// synthesize calls the edge-tts CLI for a single voice/text segment and
// returns the path to a temp MP3 file. edge-tts is an unofficial wrapper
// around Microsoft Edge's read-aloud service, so - unlike a paid Azure
// Cognitive Services key - it has no documented rate-limit guarantee; retry
// a few times with backoff before giving up.
func (c *TTSClient) synthesize(ctx context.Context, voice, ratePercent, text string) (string, error) {
	tmp, err := os.CreateTemp("", "tts-segment-*.mp3")
	if err != nil {
		return "", err
	}
	tmp.Close()

	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		cmd := exec.CommandContext(ctx, "edge-tts",
			"--voice", voice,
			"--rate="+ratePercent,
			"--text", text,
			"--write-media", tmp.Name(),
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("edge-tts (voice %s): %w: %s", voice, err, stderr.String())
			continue
		}
		// edge-tts occasionally exits 0 but writes an empty/truncated file
		// under rapid consecutive requests (no documented rate-limit
		// contract, unlike a paid Azure key) - treat that as a failed
		// attempt too, not just a nonzero exit code.
		if info, statErr := os.Stat(tmp.Name()); statErr != nil || info.Size() == 0 {
			lastErr = fmt.Errorf("edge-tts (voice %s): wrote empty output, text=%q stderr=%s", voice, text, stderr.String())
			continue
		}
		return tmp.Name(), nil
	}
	os.Remove(tmp.Name())
	return "", lastErr
}

func transcodeToMP3(wavPath, mp3Path string) error {
	cmd := exec.Command("ffmpeg", "-i", wavPath, "-codec:a", "libmp3lame", "-qscale:a", "2", "-y", mp3Path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}
	return nil
}

type segment struct {
	voice       string
	ratePercent string
	pauseMs     int
	text        string
}

func encodeSegment(voice, ratePercent string, pauseMs int, text string) string {
	return strings.Join([]string{voice, ratePercent, strconv.Itoa(pauseMs), text}, segFieldDelim) + segDelim
}

func decodeSegments(query string) []segment {
	var segments []segment
	for _, raw := range strings.Split(query, segDelim) {
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, segFieldDelim, 4)
		if len(parts) != 4 {
			continue
		}
		pauseMs, _ := strconv.Atoi(parts[2])
		segments = append(segments, segment{
			voice:       parts[0],
			ratePercent: parts[1],
			pauseMs:     pauseMs,
			text:        parts[3],
		})
	}
	return segments
}

// parsePauseMs converts a pause value like "1000ms" or "0.0" (both forms are
// used by callers - see PrepareQuery's silence value passthrough in the
// previous Azure client) into whole milliseconds.
func parsePauseMs(pause string) int {
	pause = strings.TrimSuffix(strings.TrimSpace(pause), "ms")
	ms, err := strconv.ParseFloat(pause, 64)
	if err != nil {
		return 0
	}
	return int(ms)
}

func (c *TTSClient) PrepareQueryWithRandomVoice(text, pause string, addSplitAudio bool) string {
	speaker := c.GetRandomVoice()
	return c.PrepareQuery(text, speaker, pause, addSplitAudio)
}

// PrepareQueryRaw builds a single-voice query without stripping whitespace
// from text, so spaces are rendered as pauses between words by edge-tts's
// underlying Azure neural engine (the same behavior the Azure Cognitive
// Services SSML endpoint had).
func (c *TTSClient) PrepareQueryRaw(text, speaker, pause string) string {
	slog.Debug("prepare tts raw query", "voice", speaker, "text", text)
	return encodeSegment(speaker, zhRatePercent, parsePauseMs(pause), text)
}

func (c *TTSClient) PrepareEnglishQuery(text, pause string) string {
	speaker := "en-US-AvaMultilingualNeural"
	slog.Debug("prepare tts en query", "voice", speaker, "text", text)
	return encodeSegment(speaker, enRatePercent, parsePauseMs(pause), text)
}

// if text contains whitespaces and addSplitAudio is true, text is added twice, once with all
// whitespaces stipped off and once with whitespaces. the underlying tts engine renders whitespaces as pauses in the audio.
func (c *TTSClient) PrepareQuery(text, speaker, pause string, addSplitAudio bool) string {
	slog.Debug("prepare tts query", "voice", speaker, "text", text)
	pauseMs := parsePauseMs(pause)
	query := encodeSegment(speaker, zhRatePercent, pauseMs, strings.ReplaceAll(text, " ", ""))
	if addSplitAudio {
		query += encodeSegment(speaker, zhRatePercent, pauseMs, text)
	}
	return query
}

func contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
