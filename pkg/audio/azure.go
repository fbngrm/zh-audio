package audio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

// speed
const rate = "0.7"

type AzureClient struct {
	endpoint    string
	apiKey      string
	AudioDir    string
	ignoreChars []string
}

func NewAzureClient(apiKey, endpoint, dir string, ignoreChars []string) (*AzureClient, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "zh")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}
	return &AzureClient{
		endpoint:    endpoint,
		apiKey:      apiKey,
		AudioDir:    dir,
		ignoreChars: ignoreChars,
	}, nil
}

// we support 4 different voices only
var Voices = []string{
	"zh-CN-XiaoxiaoNeural", // female
	"zh-CN-YunjianNeural",  // male
	"zh-CN-XiaochenNeural", // female
	// "zh-CN-YinyangNeural",  // male / broken
	"zh-CN-YunyiMultilingualNeural", // male
}

func (c *AzureClient) GetRandomVoice() string {
	rand.Seed(time.Now().UnixNano()) // initialize global pseudo random generator
	return Voices[rand.Intn(len(Voices))]
}

func (c *AzureClient) GetVoices(speakers map[string]struct{}) map[string]string {
	v := make(map[string]string)
	var i int
	for speaker := range speakers {
		v[speaker] = Voices[i]
		i++
	}
	return v
}

// download audio file from azure text-to-speech api if it doesn't exist in cache dir.
func (c *AzureClient) Fetch(ctx context.Context, query, filename string) (string, error) {
	if contains(c.ignoreChars, query) {
		return "", nil
	}
	if err := os.MkdirAll(c.AudioDir, os.ModePerm); err != nil {
		return "", err
	}
	lessonPath := filepath.Join(c.AudioDir, filename)

	resp, err := c.fetch(ctx, query, 0)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// fmt.Println("Response Headers:")
	// for key, values := range resp.Header {
	// 	for _, value := range values {
	// 		fmt.Printf("%s: %s\n", key, value)
	// 	}
	// }

	out, err := os.Create(lessonPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}

	slog.Info("audio content generated", "path", lessonPath)
	return lessonPath, nil
}

func (c *AzureClient) fetch(ctx context.Context, query string, retryCount int) (*http.Response, error) {
	if retryCount == 7 {
		return nil, fmt.Errorf("excceded retries for query: %s", query)
	}

	throttle := 0
	if retryCount != 0 {
		throttle = int(math.Pow(float64(throttle), float64(retryCount)))
		fmt.Printf("Quota Exceeded, retry in: %d seconds\n", throttle)
		time.Sleep(time.Duration(throttle) * time.Second)
	} else {
		time.Sleep(time.Duration(throttle) * time.Second)
		query = fmt.Sprintf(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="zh-CN">%s</speak>`, query)

	}

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer([]byte(query)))
	if err != nil {
		fmt.Printf("error creating request: %v", err)
		fmt.Println("retry...")
		return c.fetch(ctx, query, retryCount+1)
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("X-Microsoft-OutputFormat", "audio-16khz-128kbitrate-mono-mp3")
	req.Header.Set("User-Agent", "curl")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error sending request to azure text-to-speech api: %v", err)
		fmt.Println("retry...")
		return c.fetch(ctx, query, retryCount-1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Status-Code: ", resp.StatusCode)
		buf := new(strings.Builder)
		_, err = io.Copy(buf, resp.Body)
		if err != nil {
			fmt.Println(err)
		}
		s := buf.String()
		if s == "Quota Exceeded" || resp.StatusCode == http.StatusTooManyRequests {
			return c.fetch(ctx, query, retryCount+1)
		}
	}

	return resp, nil
}

func (c *AzureClient) PrepareQueryWithRandomVoice(text, pause string, addSplitAudio bool) string {
	speaker := c.GetRandomVoice()
	return c.PrepareQuery(text, speaker, pause, addSplitAudio)
}

func (c *AzureClient) PrepareEnglishQuery(text, pause string) string {
	speaker := "en-US-AvaMultilingualNeural"
	slog.Debug("prepare azure en query", "voice", speaker, "text", text)
	queryFmt := `
    <voice name="%s">
        <mstts:silence  type="Tailing-exact" value="%s"/>
        <prosody rate="%s">
		    %s
        </prosody>
    </voice>`
	return fmt.Sprintf(queryFmt, speaker, pause, "1.0", text)
}

// if text contains whitespaces and addSplitAudio is true, text is added twice, once with all
// whitespaces stipped off and once with whitespaces. azure api renders whitespaces as pauses in the audio.
func (c *AzureClient) PrepareQuery(text, speaker, pause string, addSplitAudio bool) string {
	slog.Debug("prepare azure query", "voice", speaker, "text", text)
	queryFmt := `
    <voice name="%s">
        <mstts:silence  type="Tailing-exact" value="%s"/>
        <prosody rate="%s">
		    %s
        </prosody>
    </voice>`
	query := fmt.Sprintf(queryFmt, speaker, pause, rate, strings.ReplaceAll(text, " ", ""))
	if addSplitAudio {
		query += fmt.Sprintf(queryFmt, speaker, pause, rate, text)
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

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
