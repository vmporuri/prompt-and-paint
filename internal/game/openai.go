package game

import (
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

var openaiClient = openai.NewClient(os.Getenv("OPENAI_API_KEY"))

const questionPrompt = "Generate a 1 sentence 'Apples to Apples' style prompt that would be fun to draw."

func generateQuestion(room *Room) string {
	req := openai.CompletionRequest{
		Model:     openai.GPT3Dot5TurboInstruct,
		MaxTokens: 500,
		Prompt:    questionPrompt,
		N:         1,
	}
	resp, err := openaiClient.CreateCompletion(room.Ctx, req)
	if err != nil {
		log.Printf("Completion error: %v", err)
		return ""
	}
	return resp.Choices[0].Text
}

func generatePicture(client *Client, prompt string) (string, error) {
	req := openai.ImageRequest{
		Model:          openai.CreateImageModelDallE2,
		Prompt:         prompt,
		Size:           openai.CreateImageSize256x256,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              1,
	}
	resp, err := openaiClient.CreateImage(client.Ctx, req)
	if err != nil {
		log.Printf("Error generating image: %v", err)
		return "", err
	}
	return resp.Data[0].URL, nil
}
