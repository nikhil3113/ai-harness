package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

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


// ─── Tool call parsing ────────────────────────────────────────────────────────

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


