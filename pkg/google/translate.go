package google

import (
	"context"
	"fmt"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
)

func Translate(text string) (string, error) {
	ctx := context.Background()

	client, err := translate.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	translations, err := client.Translate(ctx,
		[]string{text},
		language.English,
		&translate.Options{
			Source: language.Chinese,
			Format: translate.Text,
		})
	if err != nil {
		return "", fmt.Errorf("translate: %w", err)
	}
	if len(translations) == 0 {
		return "", fmt.Errorf("translate returned empty response to text: %s", text)
	}
	fmt.Println(translations)
	return translations[0].Text, nil
}
