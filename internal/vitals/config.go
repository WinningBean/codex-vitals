package vitals

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Model  string
	Effort string
	MTime  time.Time
}

func DefaultCodexHome(homeDir string) string {
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		return codexHome
	}
	return filepath.Join(homeDir, ".codex")
}

func LoadConfig(codexHome string) (Config, error) {
	path := filepath.Join(codexHome, "config.toml")
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return Config{}, err
	}

	config := Config{MTime: info.ModTime()}
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(stripInlineComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			continue
		}
		if section != "" {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		parsed := parseTomlString(value)
		switch key {
		case "model":
			config.Model = parsed
		case "model_reasoning_effort":
			config.Effort = parsed
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func stripInlineComment(line string) string {
	var quote rune
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if quote == '"' && r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		if r == '#' {
			return line[:i]
		}
	}
	return line
}

func parseTomlString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, `"`) {
		parsed, err := strconv.Unquote(value)
		if err == nil {
			return parsed
		}
	}
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return value[1 : len(value)-1]
	}
	return value
}

func SelectModel(turn ModelInfo, config Config) ModelInfo {
	if !turn.HasTurn {
		return ModelInfo{
			Model:  config.Model,
			Effort: config.Effort,
		}
	}
	if !config.MTime.IsZero() && !turn.Timestamp.IsZero() && config.MTime.After(turn.Timestamp) {
		selected := turn
		if config.Model != "" {
			selected.Model = config.Model
		}
		if config.Effort != "" {
			selected.Effort = config.Effort
		}
		return selected
	}
	return turn
}

func SelectModelForContextMode(turn ModelInfo, config Config, _ ContextMode) ModelInfo {
	// Model selection is independent of the context-% mode: SelectModel already
	// prefers config.toml when it changed (via /model) after the last turn, so
	// the model reflects /model live instead of the stale per-turn record.
	return SelectModel(turn, config)
}
