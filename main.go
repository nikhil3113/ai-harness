package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ─── ANSI colors ──────────────────────────────────────────────────────────────

const (
	colorYou       = "\033[94m" // blue
	colorAssistant = "\033[93m" // yellow
	colorTool      = "\033[92m" // green
	colorError     = "\033[91m" // red
	colorDim       = "\033[2m"  // dim
	colorReset     = "\033[0m"
	colorBold      = "\033[1m"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens,omitempty"`
	Messages  []Message `json:"messages"` 
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"` 	} `json:"error,omitempty"`
}

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

// ─── Tool implementations ─────────────────────────────────────────────────────

func resolveAbsPath(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	cwd, _ := os.Getwd()
	return filepath.Clean(filepath.Join(cwd, p))
}

func readFileTool(filename string) map[string]any {
	full := resolveAbsPath(filename)
	data, err := os.ReadFile(full)
	if err != nil {
		return map[string]any{"error": err.Error(), "file_path": full}
	}
	return map[string]any{"file_path": full, "content": string(data)}
}

func listFilesTool(path string) map[string]any {
	full := resolveAbsPath(path)
	entries, err := os.ReadDir(full)
	if err != nil {
		return map[string]any{"error": err.Error(), "path": full}
	}
	files := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		files = append(files, map[string]string{"filename": e.Name(), "type": kind})
	}
	return map[string]any{"path": full, "files": files}
}

func editFileTool(path, oldStr, newStr string) map[string]any {
	full := resolveAbsPath(path)

	if oldStr == "" {
		// FIX: use MkdirAll (not Mkdir) so it doesn't error if the dir already exists
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return map[string]any{"error": err.Error(), "path": full}
		}
		if err := os.WriteFile(full, []byte(newStr), 0644); err != nil {
			return map[string]any{"error": err.Error(), "path": full}
		}
		return map[string]any{"path": full, "action": "created_file"}
	}

	
	original, err := os.ReadFile(full)
	if err != nil {
		return map[string]any{"error": err.Error(), "path": full}
	}
	orig := string(original)
	if !strings.Contains(orig, oldStr) {
		return map[string]any{"path": full, "error": "old_str not found in file"}
	}
	edited := strings.Replace(orig, oldStr, newStr, 1)
	if err := os.WriteFile(full, []byte(edited), 0644); err != nil {
		return map[string]any{"error": err.Error(), "path": full}
	}
	return map[string]any{"path": full, "action": "edited"}
}

// ─── System prompt ────────────────────────────────────────────────────────────

const systemPrompt = `You are a coding assistant whose goal it is to help solve coding tasks.
You have access to three tools you can call at any time:

TOOL
===
Name: read_file
Description: Gets the full content of a file.
Signature: read_file({"filename": "<path>"})
===============

TOOL
===
Name: list_files
Description: Lists all files and directories at a given path.
Signature: list_files({"path": "<path>"})
===============

TOOL
===
Name: edit_file
Description: Replaces the first occurrence of old_str with new_str in a file.
             If old_str is empty, creates (or overwrites) the file with new_str.
Signature: edit_file({"path": "<path>", "old_str": "<old>", "new_str": "<new>"})
===============

Rules:
- When you want to call a tool, reply with EXACTLY one line in this format and nothing else:
    tool: TOOL_NAME({"key": "value"})
- Use compact single-line JSON with double-quoted keys.
- After receiving a tool_result(...) message, continue the task.
- Chain multiple tool calls one at a time (one tool per reply).
- If no tool is needed, respond normally.
`

// ─── Tool call parsing ────────────────────────────────────────────────────────

type ToolCall struct {
	Name string
	Args map[string]any
}

func extractToolCall(text string) []ToolCall {
	var calls []ToolCall
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "tool:") {
			continue
		}
		after := strings.TrimSpace(line[len("tool:"):])
		paren := strings.Index(after, "(")
		if paren == -1 || !strings.HasSuffix(after, ")") {
			continue
		}
		name := strings.TrimSpace(after[:paren])
		jsonStr := after[paren+1 : len(after)-1]
		var args map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &args); err != nil {
			continue
		}
		calls = append(calls, ToolCall{Name: name, Args: args})
	}
	return calls
}

func strArgs(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func executeTool(call ToolCall) map[string]any {
	switch call.Name {
	case "read_file":
		return readFileTool(strArgs(call.Args, "filename"))
	case "list_files":
		return listFilesTool(strArgs(call.Args, "path"))
	case "edit_file":
		return editFileTool(
			strArgs(call.Args, "path"),
			strArgs(call.Args, "old_str"),
			strArgs(call.Args, "new_str"),
		)
	default:
		return map[string]any{"error": "unknown tool: " + call.Name}
	}
}

// ─── Agent loop ───────────────────────────────────────────────────────────────

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

// ─── main / Cobra ─────────────────────────────────────────────────────────────

func main() {
	var (
		apiKey    string
		baseURL   string
		model     string
		maxTokens int
		verbose   bool
	)

	rootCmd := &cobra.Command{
		Use:   "aiharness",
		Short: "Minimal AI coding agent CLI — Go + Cobra",
		Long: `aiharness is a minimal AI coding agent built in Go with Cobra.

Implements the agentic loop from:
  https://www.mihaileric.com/The-Emperor-Has-No-Clothes/

Uses the OpenAI-compatible /chat/completions API — works with any provider:
  OpenAI, Anthropic, OpenRouter, Google Gemini, Groq, Ollama, Mistral, Together AI, etc.`,
	}

	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&apiKey, "api-key", "k", "", "API key (default: $API_KEY)")
	pf.StringVarP(&baseURL, "base-url", "u", "https://openrouter.ai/api/v1", "OpenAI-compatible base URL")
	pf.StringVarP(&model, "model", "m", "meta-llama/llama-3.3-8b-instruct:free", "Model name")
	pf.IntVarP(&maxTokens, "max-tokens", "t", 2000, "Max tokens per LLM response")
	pf.BoolVarP(&verbose, "verbose", "v", false, "Show raw tool results and HTTP details")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Start the interactive coding agent",
		Long: `Start the interactive agentic REPL.

Type natural language — the agent will read, list, and edit files
on your local filesystem by calling the LLM.

Examples:
  # OpenRouter free model (default)
  aiharness run --api-key $OPENROUTER_KEY

  # OpenAI
  aiharness run --base-url https://api.openai.com/v1 --model gpt-4o --api-key $OPENAI_KEY

  # Google Gemini
  aiharness run --base-url https://generativelanguage.googleapis.com/v1beta/openai --model gemini-2.0-flash --api-key $GEMINI_KEY

  # Groq (free tier)
  aiharness run --base-url https://api.groq.com/openai/v1 --model llama-3.3-70b-versatile --api-key $GROQ_KEY

  # Anthropic
  aiharness run --base-url https://api.anthropic.com/v1 --model claude-sonnet-4-20250514 --api-key $ANTHROPIC_KEY

  # Ollama (local, no key needed)
  aiharness run --base-url http://localhost:11434/v1 --model llama3.2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key := firstNonEmpty(apiKey, os.Getenv("API_KEY"))
			url := firstNonEmpty(baseURL, os.Getenv("LLM_BASE_URL"))
			if url == "" {
				return fmt.Errorf("--base-url is required (or set $LLM_BASE_URL)")
			}
			runAgentLoop(url, key, model, maxTokens, verbose)
			return nil
		},
	}

	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "List available tools and their call signatures",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("\n%s%sAvailable Tools%s\n\n", colorBold, colorTool, colorReset)
			type toolInfo struct{ name, sig, desc string }
			list := []toolInfo{
				{
					"read_file",
					`read_file({"filename": "<path>"})`,
					"Read the full contents of a file",
				},
				{
					"list_files",
					`list_files({"path": "<dir>"})`,
					"List all files and directories under a path",
				},
				{
					"edit_file",
					`edit_file({"path":"<path>","old_str":"<old>","new_str":"<new>"})`,
					"Replace first occurrence of old_str with new_str.\n                 Empty old_str = create/overwrite the file.",
				},
			}
			for i, t := range list {
				fmt.Printf("  %s%d. %s%s\n", colorBold, i+1, t.name, colorReset)
				fmt.Printf("     %sSig: %s%s\n", colorDim, t.sig, colorReset)
				fmt.Printf("     Desc: %s\n\n", t.desc)
			}
		},
	}

	providersCmd := &cobra.Command{
		Use:   "providers",
		Short: "Show example provider base URLs and models",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("\n%s%sOpenAI-Compatible Providers%s\n\n", colorBold, colorTool, colorReset)
			type provider struct{ name, url, model, key string }
			list := []provider{
				{
					"OpenRouter (free)",
					"https://openrouter.ai/api/v1",
					"tencent/hy3-preview:free",
					"OPENROUTER_API_KEY",
				},
				{
					"Google Gemini",
					"https://generativelanguage.googleapis.com/v1beta/openai",
					"gemini-2.0-flash",
					"GEMINI_API_KEY",
				},
				{
					"Groq",
					"https://api.groq.com/openai/v1",
					"llama-3.3-70b-versatile",
					"GROQ_API_KEY",
				},
				{
					"Anthropic",
					"https://api.anthropic.com/v1",
					"claude-sonnet-4-20250514",
					"ANTHROPIC_API_KEY",
				},
				{
					"OpenAI",
					"https://api.openai.com/v1",
					"gpt-4o-mini",
					"OPENAI_API_KEY",
				},
				{
					"Mistral",
					"https://api.mistral.ai/v1",
					"mistral-small-latest",
					"MISTRAL_API_KEY",
				},
				{
					"Together AI",
					"https://api.together.xyz/v1",
					"meta-llama/Llama-3-8b-chat-hf",
					"TOGETHER_API_KEY",
				},
				{
					"Ollama (local)",
					"http://localhost:11434/v1",
					"llama3.2",
					"(none needed)",
				},
			}
			for _, p := range list {
				fmt.Printf("  %s%s%s\n", colorBold, p.name, colorReset)
				fmt.Printf("    %s--base-url%s  %s\n", colorDim, colorReset, p.url)
				fmt.Printf("    %s--model%s     %s\n", colorDim, colorReset, p.model)
				fmt.Printf("    %skey env%s     $%s\n\n", colorDim, colorReset, p.key)
			}
		},
	}

	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "List example model IDs per provider",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("\n%sRun `aiharness providers` for full provider list with URLs.%s\n\n", colorDim, colorReset)
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version info",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("aiharness v1.0.0")
			fmt.Println("Built with: Go + github.com/spf13/cobra")
			fmt.Println("API:        OpenAI-compatible /chat/completions")
			fmt.Println("Reference:  https://www.mihaileric.com/The-Emperor-Has-No-Clothes/")
		},
	}

	rootCmd.AddCommand(runCmd, toolsCmd, providersCmd, modelsCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
