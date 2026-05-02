package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ─── callLLM — works with ANY OpenAI-compatible provider ─────────────────────
//
// Provider examples:
//   Anthropic (via openai-compat)  https://api.anthropic.com/v1          claude-sonnet-4-20250514
//   OpenAI                         https://api.openai.com/v1             gpt-4o
//   OpenRouter (free models)       https://openrouter.ai/api/v1          meta-llama/llama-3.3-8b-instruct:free
//   Google Gemini                  https://generativelanguage.googleapis.com/v1beta/openai  gemini-2.0-flash
//   Groq                           https://api.groq.com/openai/v1        llama-3.3-70b-versatile
//   Ollama (local)                 http://localhost:11434/v1              llama3.2
//   Together AI                    https://api.together.xyz/v1           meta-llama/Llama-3-8b-chat-hf
//   Mistral                        https://api.mistral.ai/v1             mistral-small-latest

func callLLM(baseURL, apiKey, model string, maxTokens int, messages []Message) (string, error) {
	req := ChatRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  messages,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request build error: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, raw)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return "", fmt.Errorf("JSON decode error: %w\nbody: %s", err, raw)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty choices in response: %s", raw)
	}
	return chatResp.Choices[0].Message.Content, nil
}


