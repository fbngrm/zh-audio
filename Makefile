.PHONY: audio
src=./audio
dst=./audio/concat
audio:
	go run cmd/main.go -s $(s) -type $(type)
	mkdir -p $(dst)
	cd $(src); for i in *.mp3; do ffmpeg -i "$$i" -filter:a "atempo=0.7" /tmp/"$${i%.*}_slowed.mp3"; done
	cd $(src); for i in *.mp3; do ffmpeg -i  /tmp/"$${i%.*}_slowed.mp3" -af "apad=pad_dur=1"  /tmp/"$${i%.*}_slowed_silence.mp3"; done
	cd $(src); for i in *.mp3; do ffmpeg -i  "$$i" -af "apad=pad_dur=1"  /tmp/"$${i%.*}_silence.mp3"; done
	cd $(src); for i in *.mp3; do ffmpeg -i /tmp/"$${i%.*}_silence.mp3" -i /tmp/"$${i%.*}_slowed_silence.mp3" -filter_complex "[0:a][1:a][0:a][1:a][0:a][1:a]concat=n=6:v=0:a=1[out]" -map "[out]" ./concat/"$${i%.*}.mp3"; done
