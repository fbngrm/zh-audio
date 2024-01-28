src_zh=./out/zh
src_slow=./out/slow
src_en=../en
dst=./out/concat

.PHONY: d
d: clean dialogs audio

.PHONY: dialogs
dialogs:
	go run cmd/main.go -src $(src) -d

.PHONY: s
s: clean sentences audio

.PHONY: sentences
sentences:
	go run cmd/main.go -src $(src)

.PHONY: audio
audio:
	mkdir -p $(dst)
	cd $(src_zh); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/zh/"$${i%.*}_silence.mp3"; done
	cd $(src_slow); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/zh/slow/"$${i%.*}_silence.mp3"; done
	cd $(src_zh); for i in *.mp3; do ffmpeg -i $(src_en)/"$$i" -i /tmp/zh/"$${i%.*}_silence.mp3" -i /tmp/zh/slow/"$${i%.*}_silence.mp3" -i ../../"peep_silence.mp3" -filter_complex "[1:a][2:a][1:a][2:a][0:a][1:a][2:a][3:a]concat=n=8:v=0:a=1[out]" -map "[out]" ../concat/"$${i%.*}.mp3"; done

.PHONY: clean
clean:
	rm -r /tmp/zh || true
	mkdir -p /tmp/zh/slow
