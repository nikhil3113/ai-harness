package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func runAgentLoop(baseURL, apiKey, model string, maxTokens int, verbose bool) {
	scanner := bufio.NewScanner(os.Stdin)

	// System prompt injected as first message (OpenAI-compat style)
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}

	fmt.Printf("\n%s%s", colorBold, colorTool)
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║   AI Coding Harness  ·  Go + Cobra CLI   ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("%s", colorReset)
	fmt.Printf("%sbase_url:   %s%s\n", colorDim, baseURL, colorReset)
	fmt.Printf("%smodel:      %s · max_tokens: %d%s\n", colorDim, model, maxTokens, colorReset)
	fmt.Printf("%stools:      read_file · list_files · edit_file%s\n", colorDim, colorReset)
	fmt.Printf("%sCtrl+C or Ctrl+D to exit%s\n\n", colorDim, colorReset)

	for {
		fmt.Printf("%s%sYou ›%s ", colorBold, colorYou, colorReset)
		if !scanner.Scan() {
			break
		}
		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		messages = append(messages, Message{Role: "user", Content: userInput})

		// Inner agentic loop — keep going until LLM stops calling tools
		for {
			if verbose {
				fmt.Printf("%s  [POST %s/chat/completions model=%s]%s\n", colorDim, baseURL, model, colorReset)
			} else {
				fmt.Printf("%s  thinking...%s\n", colorDim, colorReset)
			}

			response, err := callLLM(baseURL, apiKey, model, maxTokens, messages)
			if err != nil {
				fmt.Printf("%sError: %v%s\n", colorError, err, colorReset)
				break
			}

			toolCalls := extractToolCall(response)

			if len(toolCalls) == 0 {
				fmt.Printf("\n%s%sAssistant ›%s %s\n\n", colorBold, colorAssistant, colorReset, response)
				messages = append(messages, Message{Role: "assistant", Content: response})
				break
			}

			messages = append(messages, Message{Role: "assistant", Content: response})
			for _, call := range toolCalls {
				argsJSON, _ := json.Marshal(call.Args)
				fmt.Printf("%s%s  ⚙  %s(%s)%s\n", colorBold, colorTool, call.Name, argsJSON, colorReset)
				result := executeTool(call)
				resultJSON, _ := json.Marshal(result)
				if verbose {
					fmt.Printf("%s     → %s%s\n", colorDim, resultJSON, colorReset)
				}
				toolMsg := fmt.Sprintf("tool_result(%s)", string(resultJSON))
				messages = append(messages, Message{Role: "user", Content: toolMsg})
				fmt.Printf("%s     ✓ done%s\n", colorTool, colorReset)
			}
		}
	}

	fmt.Printf("\n%sBye!%s\n", colorDim, colorReset)
}
