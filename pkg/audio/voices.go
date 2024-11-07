package audio

import (
	"math/rand"
	"time"

	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

var voicesEN = []*texttospeechpb.VoiceSelectionParams{
	{
		LanguageCode: "en-US",
		Name:         "en-US-Polyglot-1",
		SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
	},
	{
		LanguageCode: "en-US",
		Name:         "en-US-Standard-J",
		SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
	},
	{
		LanguageCode: "en-US",
		Name:         "en-US-Studio-O",
		SsmlGender:   texttospeechpb.SsmlVoiceGender_FEMALE,
	},
}

// we support 4 different voices only
var VoicesZH = []*texttospeechpb.VoiceSelectionParams{
	{
		LanguageCode: "cmn-CN",
		Name:         "cmn-CN-Wavenet-C",
		SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
	},
	{
		LanguageCode: "cmn-CN",
		Name:         "cmn-CN-Wavenet-A",
		SsmlGender:   texttospeechpb.SsmlVoiceGender_FEMALE,
	},
	{
		LanguageCode: "cmn-CN",
		Name:         "cmn-TW-Wavenet-C",
		SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
	},
	{
		LanguageCode: "cmn-CN",
		Name:         "cmn-TW-Wavenet-A",
		SsmlGender:   texttospeechpb.SsmlVoiceGender_FEMALE,
	},
}

func GetVoicesZH(speakers map[string]struct{}) map[string]*texttospeechpb.VoiceSelectionParams {
	v := make(map[string]*texttospeechpb.VoiceSelectionParams)
	var i int
	for speaker := range speakers {
		v[speaker] = VoicesZH[i]
		i++
	}
	return v
}

func GetRandomVoiceZH() *texttospeechpb.VoiceSelectionParams {
	rand.Seed(time.Now().UnixNano()) // initialize global pseudo random generator
	return VoicesZH[rand.Intn(len(VoicesZH))]
}

func GetRandomVoiceEN() *texttospeechpb.VoiceSelectionParams {
	rand.Seed(time.Now().UnixNano()) // initialize global pseudo random generator
	return voicesEN[rand.Intn(len(voicesEN))]
}
