package ai

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/genai"
)

func callGeminiWithRetry(appCtx context.Context, agent *genai.Client, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second
	const timeoutInterval = 3 * time.Minute

	timeoutCtx, _ := context.WithTimeout(context.Background(), timeoutInterval)
	var lastError error

	for i := 0; i < maxRetries; i++ {
		resp, err := agent.Models.GenerateContent(timeoutCtx, model, contents, config)

		if err == nil {
			return resp, nil
		}
		lastError = err
		log.Printf("Gemini API failed (Attempt %d/%d): %v", i+1, maxRetries, err)

		select {
		case <-appCtx.Done():
			return nil, fmt.Errorf("shutdown during retry: %w", appCtx.Err())
		default:
		}

		if i < 2 {
			select {
			case <-time.After(2 * time.Second):
			case <-appCtx.Done():
				return nil, appCtx.Err()
			}
		}
	}

	return nil, lastError
}

func SummarizeVideo(ctx context.Context, agent *genai.Client, videoURL string) (output string, err error) {
	if agent == nil {
		return "", fmt.Errorf("Agent is nil")
	}
	if videoURL == "" {
		return "", fmt.Errorf("Invalid URL")
	}

	log.Println("Processing video")

	prompts := `
	Summarize video in Traditional Chinese.
	Logic:
	- T <= 10m -> 3-5 pts
	- T > 30m -> 5+ pts
	Output: Markdown bullets. Focus on high-value info. With timestamps`
	parts := []*genai.Part{
		genai.NewPartFromURI(videoURL, "video/mp4"),
		genai.NewPartFromText(prompts),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	config := &genai.GenerateContentConfig{
		MediaResolution: genai.MediaResolutionLow,
	}

	modelName := "gemini-3-flash-preview"

	resp, err := callGeminiWithRetry(ctx, agent, modelName, contents, config)

	if err != nil {
		return "", err
	}
	if len(resp.Candidates) > 0 {
		for _, part := range resp.Candidates[0].Content.Parts {
			output += part.Text
		}
	}
	return output, nil
}

func InitGemini(ctx context.Context) (agent *genai.Client, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	key := os.Getenv("GOOGLE_GEMINI_API")

	agent, err = genai.NewClient(timeoutCtx, &genai.ClientConfig{
		APIKey: key,
	})
	if err != nil {
		return nil, err
	}

	return agent, err
}
