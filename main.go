package main

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)


const (
	colorYou       = "\033[94m" // blue
	colorAssistant = "\033[93m" // yellow
	colorTool      = "\033[92m" // green
	colorError     = "\033[91m" // red
	colorDim       = "\033[2m"  // dim
	colorReset     = "\033[0m"
	colorBold      = "\033[1m"
)



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
