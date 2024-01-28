package deepl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Client struct {
	endpoint   string
	apiKey     string
	targetLang string
}

func NewClient(apiKey string) *Client {
	return &Client{
		endpoint:   "https://api-free.deepl.com/v2/translate",
		apiKey:     apiKey,
		targetLang: "EN",
	}
}

type Request struct {
	Text       []string `json:"text"`
	TargetLang string   `json:"target_lang"`
}

type Translation struct {
	Text string `json:"text"`
}

type Response struct {
	Translations []Translation `json:"translations"`
}

func (c *Client) Translate(query []string, retryCount int) ([]Translation, error) {
	if retryCount == 0 {
		return nil, errors.New("max retries for DeepL API")
	}
	payload := Request{
		Text:       query,
		TargetLang: "EN",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("error marshalling JSON payload: %v", err)
		fmt.Println("retry...")
		return c.Translate(query, retryCount-1)
	}

	// fmt.Println(string(jsonPayload))
	// Set up the request object
	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("error creating request: %v", err)
		fmt.Println("retry...")
		return c.Translate(query, retryCount-1)
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "DeepL-Auth-Key "+c.apiKey)

	// fmt.Println(req)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error sending request: %v", err)
		fmt.Println("retry...")
		return c.Translate(query, retryCount-1)
	}
	defer resp.Body.Close()

	// Read the response body
	// var result map[string]interface{}
	var result Response
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("error decoding JSON response: %v\n", err)
		fmt.Println("retry...")
		return c.Translate(query, retryCount-1)
	}

	if len(result.Translations) == 0 {
		fmt.Println("no result, retry...")
		return c.Translate(query, retryCount-1)
	}
	return result.Translations, nil
}
