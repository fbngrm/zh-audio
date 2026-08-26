// Command tingting-stories generates audio for the tingting app's per-word
// "story" (a short multi-sentence narrative using the word), stored in
// ../../tingting/data/hsk{N}.json as a "story" array of sentence objects.
//
// Unlike tingting-sentences, this only generates the continuous ("normal")
// Chinese reading - no segmented (word-paused) variant and no English
// narration.
//
// Usage:
//
//	go run ./cmd/tingting-stories
//	go run ./cmd/tingting-stories -level hsk1
//	go run ./cmd/tingting-stories -workers 16
//	go run ./cmd/tingting-stories -force
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fbngrm/zh-audio/pkg/audio"
)

const zhVoice = "zh-CN-XiaoxiaoNeural"

var hskLevels = []string{"hsk1", "hsk2", "hsk3", "hsk4", "hsk5", "hsk6"}

var ignoreChars = []string{"!", "！", "？", "?", "，", ",", ".", "。", "", " ", "、"}

type storyItem struct {
	Sentence string `json:"sentence"`
}

type wordEntry struct {
	No    int         `json:"no"`
	Word  string      `json:"word"`
	Story []storyItem `json:"story"`
}

type job struct {
	level    string
	stem     string // e.g. "爱_story_0"
	sentence string
	outDir   string
}

func main() {
	dataDir := flag.String("data", "../tingting/data", "tingting data dir containing hsk{N}.json")
	outDir := flag.String("out", "../tingting/assets/audio/stories", "output dir, organized by level subdir")
	level := flag.String("level", "", "only this level, e.g. hsk1 (default: all)")
	workers := flag.Int("workers", 8, "concurrent workers")
	limit := flag.Int("limit", 0, "limit to first N words per level, 0 = no limit (for testing)")
	force := flag.Bool("force", false, "regenerate even if output files already exist")
	scratch := flag.String("scratch", "./out/tingting-stories-scratch", "scratch dir for the tts client (wiped on start; must be on the same filesystem as -out)")
	flag.Parse()

	tts, err := audio.NewTTSClient(*scratch, ignoreChars, true)
	if err != nil {
		log.Fatal(err)
	}

	levels := hskLevels
	if *level != "" {
		levels = []string{*level}
	}

	for _, lvl := range levels {
		if err := processLevel(tts, *dataDir, *outDir, lvl, *workers, *limit, *force); err != nil {
			log.Printf("%s: %v", lvl, err)
		}
	}
}

func processLevel(tts *audio.TTSClient, dataDir, outDir, level string, workers, limit int, force bool) error {
	path := filepath.Join(dataDir, level+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%s: data file not found, skipping\n", level)
			return nil
		}
		return err
	}
	var words []wordEntry
	if err := json.Unmarshal(raw, &words); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	levelOutDir := filepath.Join(outDir, level)
	if err := os.MkdirAll(levelOutDir, os.ModePerm); err != nil {
		return err
	}

	// Same duplicate-word collision guard as tingting-sentences: a few
	// levels have two dictionary entries sharing identical word text but
	// different readings (e.g. 重 zhòng vs chóng) - disambiguate with the
	// entry's unique "no" field, but only when the word actually repeats.
	wordCounts := map[string]int{}
	for _, w := range words {
		if w.Word != "" {
			wordCounts[w.Word]++
		}
	}

	var jobs []job
	count := 0
	for _, w := range words {
		if w.Word == "" {
			continue
		}
		if limit > 0 && count >= limit {
			break
		}
		count++
		stemWord := w.Word
		if wordCounts[w.Word] > 1 {
			stemWord = fmt.Sprintf("%s_no%d", w.Word, w.No)
		}
		for i, item := range w.Story {
			if item.Sentence == "" {
				continue
			}
			jobs = append(jobs, job{
				level:    level,
				stem:     fmt.Sprintf("%s_story_%d", stemWord, i),
				sentence: item.Sentence,
				outDir:   levelOutDir,
			})
		}
	}

	pending := jobs[:0:0]
	for _, j := range jobs {
		if force || !exists(normalPath(j)) {
			pending = append(pending, j)
		}
	}

	fmt.Printf("\n%s: %d words, %d story sentences, %d already done, %d to generate\n",
		level, count, len(jobs), len(jobs)-len(pending), len(pending))
	if len(pending) == 0 {
		return nil
	}

	var done, failed int64
	var mu sync.Mutex
	var failures []string

	jobCh := make(chan job)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				if err := processJob(tts, j, force); err != nil {
					atomic.AddInt64(&failed, 1)
					mu.Lock()
					failures = append(failures, fmt.Sprintf("%s: %v", j.stem, err))
					mu.Unlock()
					fmt.Printf("\n  FAILED [%s]: %v\n", j.stem, err)
				} else {
					atomic.AddInt64(&done, 1)
				}
				fmt.Printf("\r  %d/%d  done=%d  failed=%d  last=%-20s",
					atomic.LoadInt64(&done)+atomic.LoadInt64(&failed), len(pending),
					atomic.LoadInt64(&done), atomic.LoadInt64(&failed), j.stem)
			}
		}()
	}
	for _, j := range pending {
		jobCh <- j
	}
	close(jobCh)
	wg.Wait()
	fmt.Println()

	for _, f := range failures {
		fmt.Printf("  FAILED %s\n", f)
	}
	return nil
}

func normalPath(j job) string {
	return filepath.Join(j.outDir, j.stem+"_normal.mp3")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func processJob(tts *audio.TTSClient, j job, force bool) error {
	ctx := context.Background()
	dest := normalPath(j)
	if !force && exists(dest) {
		return nil
	}
	stripped := strings.ReplaceAll(j.sentence, " ", "")
	return fetchTo(ctx, tts, tts.PrepareQueryRaw(stripped, zhVoice, "0ms"), j.level, j.stem+"_normal", dest)
}

// fetchTo synthesizes query via the TTS client (which always writes into its
// own flat AudioDir) and moves the result to the real destination path.
func fetchTo(ctx context.Context, tts *audio.TTSClient, query, level, uniqueName, destPath string) error {
	scratchName := level + "_" + uniqueName + "_" + strconv.FormatInt(int64(os.Getpid()), 36) + ".mp3"
	tmpPath, err := tts.Fetch(ctx, query, scratchName)
	if err != nil {
		return err
	}
	if tmpPath == "" {
		return fmt.Errorf("empty result (query filtered as ignored char?)")
	}
	return moveFile(tmpPath, destPath)
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	os.Remove(src)
	return nil
}
