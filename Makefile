out_dir=./out
src_zh=$(out_dir)/zh
src_en=../en

today := $(shell date +"%Y-%m-%d")
export_dir=/home/f/Dropbox/zh/audio-loops/$(today)/
cache_dir=/home/f/Dropbox/zh/cache/audio/
loop_cache_dir=/home/f/Dropbox/zh/cache/audio-loops/

export AUDIO_CACHE_DIR=$(cache_dir)

.PHONY: w
w: clean words add-beep

.PHONY: words
words:
	go run cmd/main.go -src $(src) -w

.PHONY: d
d: clean segment-dia dialogs audio add-beep

.PHONY: dialogs
dialogs:
	go run cmd/main.go -src $(src) -d

.PHONY: s
s: clean segment-sen sentences audio-sen

.PHONY: sentences
sentences:
	go run cmd/main.go -src $(src) -s

.PHONY: c
c: clean clozes add-beep

.PHONY: clozes
clozes:
	go run cmd/main.go -src $(src) -c

.PHONY: p
p: clean patterns

.PHONY: patterns
patterns:
	go run cmd/main.go -src $(src) -p
	mkdir -p loop_cache_dir || true
	mkdir -p $(export_dir) || true
	cp -r $(out_dir)/patterns/* $(export_dir)
	cp -r $(out_dir)/patterns/* $(loop_cache_dir)
	cp -r $(src_zh)/* $(cache_dir) || true

.PHONY: add-beep
add-beep:
	mkdir -p $(loop_cache_dir)
	cd $(src_zh); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/zh/"$${i%.*}_silence.mp3"; done
	cd $(src_zh); for i in *.mp3; do ffmpeg -i /tmp/zh/"$${i%.*}_silence.mp3" -i ../../"peep_silence.mp3" -filter_complex "[0:a][1:a]concat=n=2:v=0:a=1[out]" -map "[out]" ../concat/"$${i%.*}.mp3"; done
	thunar  $(loop_cache_dir)

.PHONY: audio-sen
audio-sen:
	mkdir -p $(loop_cache_dir)
	cd $(src_zh); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/zh/"$${i%.*}_silence.mp3"; done
	cd $(src_zh); for i in *.mp3; do ffmpeg -i $(src_en)/"$$i" -i /tmp/zh/"$${i%.*}_silence.mp3" -i ../../"peep_silence.mp3" -filter_complex "[1:a][1:a][0:a][1:a][2:a]concat=n=5:v=0:a=1[out]" -map "[out]" ../concat/"$${i%.*}.mp3"; done
	thunar  $(loop_cache_dir)

.PHONY: clean
clean:
	rm -r /tmp/zh || true
	mkdir -p /tmp/zh
	rm -r out || true

.PHONY: segment-sen
segment-sen:
	rm /tmp/segmented || true
	cd ../stanford-segmenter && ./segment.sh pku $(src) UTF-8 0 > /tmp/segmented
	cat /tmp/segmented > $(src)
	cd -

.PHONY: segment-dia
segment-dia:
	cd ../stanford-segmenter
	cd ../stanford-segmenter && ./segment.sh pku ../zh-audio/dialogs UTF-8 0 > /tmp/segmented
	cat /tmp/segmented > ../zh-audio/dialogs
	cd -

.PHONY: open
open:
	@if [ -d "$(loop_cache_dir)" ]; then \
		thunar $(loop_cache_dir); \
	else \
		thunar $(src_zh); \
	fi
