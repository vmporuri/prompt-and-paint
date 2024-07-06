package game

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

type response struct {
	Prompt string `json:"prompt"`
}

const (
	maxTokens   = 75
	temperature = 1
	topP        = 1
	n           = 1
	stream      = false
)

const questionPrompt = `
	Give me one funny, 'Apples to Apples' style prompt.
	Your response should be valid JSON with the field 'prompt' with no markdown.
`

var (
	openaiClient = openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	message      = openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: questionPrompt,
	}
)

func generateAIQuestion(ctx context.Context) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT4Turbo,
		MaxTokens:   maxTokens,
		Messages:    []openai.ChatCompletionMessage{message},
		Temperature: temperature,
		TopP:        topP,
		N:           n,
		Stream:      stream,
	}
	resp, err := openaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Printf("Completion error: %v", err)
		return "", err
	}
	responseJSON := &response{}
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), responseJSON)
	if err != nil {
		log.Printf("Error marshalling OpenAI response: %v", err)
		return "", err
	}
	return responseJSON.Prompt, nil
}

func generateAIPicture(ctx context.Context, prompt string) (string, error) {
	req := openai.ImageRequest{
		Prompt:         prompt,
		Model:          openai.CreateImageModelDallE3,
		Quality:        openai.CreateImageQualityStandard,
		Size:           openai.CreateImageSize1024x1024,
		Style:          openai.CreateImageStyleNatural,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              1,
	}
	resp, err := openaiClient.CreateImage(ctx, req)
	if err != nil {
		log.Printf("Error generating image: %v", err)
		return "", err
	}
	return resp.Data[0].URL, nil
}
