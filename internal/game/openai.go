package game

import (
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

const questionPrompt = "Generate a 1 sentence 'Apples to Apples' style prompt that would be fun to draw."

func generateQuestion(room *Room) string {
	c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	req := openai.CompletionRequest{
		Model:     openai.GPT3Dot5TurboInstruct,
		MaxTokens: 500,
		Prompt:    questionPrompt,
	}
	resp, err := c.CreateCompletion(room.Ctx, req)
	if err != nil {
		log.Printf("Completion error: %v\n", err)
		return ""
	}
	return resp.Choices[0].Text
}
