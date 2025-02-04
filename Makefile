src_zh=./out/zh
src_en=../en
concat_dir=./out/concat

today := $(shell date +"%Y-%m-%d")
export_dir=~/Dropbox/zh/audio-loops/$(today)/
cache_dir=~/Dropbox/zh/cache/audio-loops/

.PHONY: w
w: clean words

.PHONY: words
words:
	go run cmd/main.go -src $(src) -w

.PHONY: d
d: clean segment-dia dialogs audio

.PHONY: dialogs
dialogs:
	go run cmd/main.go -src $(src) -d

.PHONY: s
s: clean segment-sen sentences audio

.PHONY: sentences
sentences:
	go run cmd/main.go -src $(src) -s

.PHONY: c
c: clean
	go run cmd/main.go -src $(src) -c

.PHONY: p
p: clean
	go run cmd/main.go -src $(src) -p

.PHONY: audio
audio:
	mkdir -p $(concat_dir)
	cd $(src_zh); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/zh/"$${i%.*}_silence.mp3"; done
	cd $(src_zh); for i in *.mp3; do ffmpeg -i $(src_en)/"$$i" -i /tmp/zh/"$${i%.*}_silence.mp3" -i ../../"peep_silence.mp3" -filter_complex "[1:a][1:a][0:a][1:a][2:a]concat=n=5:v=0:a=1[out]" -map "[out]" ../concat/"$${i%.*}.mp3"; done
	thunar  $(concat_dir)

.PHONY: clean
clean:
	rm -r /tmp/zh || true
	mkdir -p /tmp/zh
	rm -r out || true

.PHONY: segment-sen
segment-sen:
	rm /tmp/segmented || true
	cd ../stanford-segmenter && ./segment.sh pku ../zh-audio/sentences UTF-8 0 > /tmp/segmented
	cat /tmp/segmented > ../zh-audio/sentences
	cd -

.PHONY: segment-dia
segment-dia:
	cd ../stanford-segmenter
	cd ../stanford-segmenter && ./segment.sh pku ../zh-audio/dialogs UTF-8 0 > /tmp/segmented
	cat /tmp/segmented > ../zh-audio/dialogs
	cd -

.PHONY: open
open:
	@if [ -d "$(concat_dir)" ]; then \
		thunar $(concat_dir); \
	else \
		thunar $(src_zh); \
	fi

.PHONY: export
export:
	mkdir -p $(export_dir)
	mkdir -p $(cache_dir)
	@if [ -d "$(concat_dir)" ]; then \
		cp -r $(concat_dir)/* $(export_dir)/; \
		cp -r $(concat_dir)/* $(cache_dir)/; \
	else \
		cp -r $(src_zh)/* $(export_dir)/; \
		cp -r $(src_zh)/* $(cache_dir)/; \
	fi

