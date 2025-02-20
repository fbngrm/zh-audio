#!/bin/bash

curl --location --request POST "https://germanywestcentral.tts.speech.microsoft.com/cognitiveservices/v1" \
    --header "Ocp-Apim-Subscription-Key: 8e1e1b7d9fa34fc8b25fcf7995746e61" \
    --header "Content-Type: application/ssml+xml" \
    --header "X-Microsoft-OutputFormat: audio-16khz-128kbitrate-mono-mp3" \
    --header "User-Agent: curl" \
    --data-raw '<voice name="zh-CN-YunjianNeural"><prosody rate="0.7">会议散场了。</prosody></voice><voice name="zh-CN-YunjianNeural"><prosody rate="0.7">会议 散场 了 。</prosody></voice>' -v > debug.mp3
