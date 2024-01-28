srcZH=./out/zh
srcEN=../en
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
	cd $(srcZH); for i in *.mp3; do ffmpeg -i "$$i" -filter:a "atempo=0.7" /tmp/zh/"$${i%.*}_slowed.mp3"; done
	cd $(srcZH); for i in *.mp3; do ffmpeg -i  /tmp/zh/"$${i%.*}_slowed.mp3" -af "apad=pad_dur=1"  /tmp/zh/"$${i%.*}_slowed_silence.mp3"; done
	cd $(srcZH); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/zh/"$${i%.*}_silence.mp3"; done
	cd $(srcZH); for i in *.mp3; do ffmpeg -i $(srcEN)/"$$i" -i /tmp/zh/"$${i%.*}_silence.mp3" -i /tmp/zh/"$${i%.*}_slowed_silence.mp3" -i ../../"peep_silence.mp3" -filter_complex "[1:a][2:a][1:a][2:a][0:a][1:a][2:a][3:a]concat=n=8:v=0:a=1[out]" -map "[out]" ../concat/"$${i%.*}.mp3"; done

.PHONY: clean
clean:
	mkdir -p $(dst)
	rm -r /tmp/zh || true
	mkdir -p /tmp/zh
