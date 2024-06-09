data_dir=../zh-anki/data/$(src)/$(lesson)/input
src_zh=./out/zh
src_en=../en
dst=./out/concat

.PHONY: cp-d
cp-d: cp-dia d

.PHONY: d
d: clean segment-dia dialogs audio

.PHONY: dialogs
dialogs:
	go run cmd/main.go -src dialogs -d

.PHONY: cp-s
cp-s: cp-sen s

.PHONY: s
s: clean segment-sen sentences audio

.PHONY: sentences
sentences:
	go run cmd/main.go -src sentences

.PHONY: audio
audio:
	mkdir -p $(dst)
	cd $(src_zh); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/zh/"$${i%.*}_silence.mp3"; done
	cd $(src_zh); for i in *.mp3; do ffmpeg -i $(src_en)/"$$i" -i /tmp/zh/"$${i%.*}_silence.mp3" -i ../../"peep_silence.mp3" -filter_complex "[1:a][1:a][0:a][1:a][2:a]concat=n=5:v=0:a=1[out]" -map "[out]" ../concat/"$${i%.*}.mp3"; done

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

.PHONY: cp-sen
cp-sen:
	cp $(data_dir)/sentences .

.PHONY: cp-dia
cp-dia:
	cp $(data_dir)/dialogues .
