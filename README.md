# AI Coding Harness

Minimal AI coding agent CLI built with Go and Cobra.

Implements the agentic loop described in [The Emperor Has No Clothes](https://www.mihaileric.com/The-Emperor-Has-No-Clothes/) by Mihail Eric.

## Features

- Interactive REPL for natural-language coding tasks
- Reads, lists, and edits files via LLM tool calls
- Works with any OpenAI-compatible provider
- Built-in tool support: `read_file`, `list_files`, `edit_file`

## Install

```bash
go install github.com/anomalyco/aiharness@latest
```

Or clone and build:

```bash
git clone https://github.com/anomalyco/aiharness.git
cd aiharness
go build -o aiharness .
```

## Usage

```bash
# OpenRouter (default, free tier)
aiharness run --api-key $OPENROUTER_KEY

# OpenAI
aiharness run --base-url https://api.openai.com/v1 --model gpt-4o --api-key $OPENAI_KEY

# Google Gemini
aiharness run --base-url https://generativelanguage.googleapis.com/v1beta/openai --model gemini-2.0-flash --api-key $GEMINI_KEY

# Local Ollama (no key needed)
aiharness run --base-url http://localhost:11434/v1 --model llama3.2
```

### Available Commands

| Command      | Description                              |
|--------------|------------------------------------------|
| `run`        | Start the interactive coding agent       |
| `tools`      | List available tools and signatures      |
| `providers`  | Show example provider URLs and models    |
| `models`     | List example model IDs per provider      |
| `version`    | Show version info                        |

### Flags

| Flag            | Default                                         | Description                          |
|-----------------|-------------------------------------------------|--------------------------------------|
| `--api-key, -k` | `$API_KEY`                                      | API key for the provider             |
| `--base-url, -u`| `https://openrouter.ai/api/v1`                  | OpenAI-compatible base URL           |
| `--model, -m`   | `meta-llama/llama-3.3-8b-instruct:free`         | Model name                           |
| `--max-tokens, -t`| `2000`                                        | Max tokens per LLM response          |
| `--verbose, -v` | `false`                                         | Show raw tool results and HTTP details|

## Architecture

```
ai-harness/
├── main.go      // Entry point and CLI setup
├── types.go     // Shared structs (Message, ChatRequest, ToolCall, etc.)
├── llm.go       // LLM client and API calls
├── tools.go     // File system tools implementation
├── agent.go     // Agent loop and tool call parsing
└── commands.go  // Cobra command definitions
```

## Supported Providers

- [OpenAI](https://openai.com)
- [Anthropic](https://anthropic.com)
- [OpenRouter](https://openrouter.ai)
- [Google Gemini](https://ai.google.dev)
- [Groq](https://groq.com)
- [Mistral](https://mistral.ai)
- [Together AI](https://together.ai)
- [Ollama](https://ollama.com) (local)

## License

MIT
