#!/bin/bash

# AZURE_ENDPOINT=https://northeurope.api.cognitive.microsoft.com/
AZURE_ENDPOINT=https://northeurope.tts.speech.microsoft.com/cognitiveservices/v1
SPEECH_KEY=71hFfaZfatPk0ZePtHJdg9FbK9z0ECRCUBMKLGSEDiQl3MQ8LP0OJQQJ99BBACi5YpzXJ3w3AAAYACOGBNKw
SPEECH_REGION=northeurope

curl --location --request POST "$AZURE_ENDPOINT" \
    --header "Ocp-Apim-Subscription-Key: $SPEECH_KEY" \
    --header "Content-Type: application/ssml+xml" \
    --header "X-Microsoft-OutputFormat: audio-16khz-128kbitrate-mono-mp3" \
    --header "User-Agent: curl" \
    --data-raw '<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="zh-CN">
<voice name="zh-CN-YunyiMultilingualNeural">
        <mstts:silence  type="Tailing-exact" value="1000ms"/>
        <prosody rate="0.6" >
我
        </prosody>
    </voice>
</speak>' -v > debug.mp3
