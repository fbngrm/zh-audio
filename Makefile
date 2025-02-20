src_zh=./out/zh
src_en=../en
concat_dir=./out/concat

today := $(shell date +"%Y-%m-%d")
export_dir=~/Dropbox/zh/audio-loops/$(today)/
cache_dir=~/Dropbox/zh/cache/audio-loops/

.PHONY: w
w: clean words add-beep export

.PHONY: words
words:
	go run cmd/main.go -src $(src) -w

.PHONY: d
d: clean segment-dia dialogs audio add-beep export

.PHONY: dialogs
dialogs:
	go run cmd/main.go -src $(src) -d

.PHONY: s
s: clean segment-sen sentences audio-sen export

.PHONY: sentences
sentences:
	go run cmd/main.go -src $(src) -s

.PHONY: c
c: clean clozes add-beep export

.PHONY: clozes
clozes:
	go run cmd/main.go -src $(src) -c

.PHONY: p
p: clean patterns add-beep export

.PHONY: patterns
patterns:
	go run cmd/main.go -src $(src) -p

.PHONY: add-beep
add-beep:
	mkdir -p $(concat_dir)
	cd $(src_zh); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/zh/"$${i%.*}_silence.mp3"; done
	cd $(src_zh); for i in *.mp3; do ffmpeg -i /tmp/zh/"$${i%.*}_silence.mp3" -i ../../"peep_silence.mp3" -filter_complex "[0:a][1:a]concat=n=2:v=0:a=1[out]" -map "[out]" ../concat/"$${i%.*}.mp3"; done
	thunar  $(concat_dir)

.PHONY: audio-sen
audio-sen:
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
	@if [ -d "$(concat_dir)" ]; then \
		thunar $(concat_dir); \
	else \
		thunar $(src_zh); \
	fi

.PHONY: export
export:
	mkdir -p $(export_dir) || true
	mkdir -p $(cache_dir) || true
	@if [ -d "$(concat_dir)" ]; then \
		cp -r $(concat_dir)/* $(export_dir)/; \
		cp -r $(concat_dir)/* $(cache_dir)/; \
	else \
		cp -r $(src_zh)/* $(export_dir)/; \
		cp -r $(src_zh)/* $(cache_dir)/; \
	fi

