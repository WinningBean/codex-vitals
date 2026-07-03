# codex-vitals

The Codex HUD whose readouts actually match Codex.

`codex-vitals` is a small Go HUD for OpenAI Codex CLI sessions. Phase 1 renders
a single stdout line with the current model, git branch, working directory,
effective context-window usage, and 5-hour / weekly rate-limit remaining
percentages.

## Build

```sh
CGO_ENABLED=0 go build ./cmd/codex-vitals
```

## Run

```sh
go run ./cmd/codex-vitals -once
go run ./cmd/codex-vitals
```

To compare against a local HUD build that uses Codex TUI's current
`input_tokens + cached_input_tokens` basis, render with:

```sh
go run ./cmd/codex-vitals -once -context-mode current-hud -style current-hud
```

For a multi-line terminal footer with emojis and a context bar:

```sh
go run ./cmd/codex-vitals -once -context-mode current-hud -style answer-footer
```

By default it reads:

- rollout JSONL files under `~/.codex/sessions`
- config from `$CODEX_HOME/config.toml` or `~/.codex/config.toml`

## Accuracy

Codex reserves `12000` baseline tokens. `codex-vitals` subtracts that baseline
from both `last_token_usage.total_tokens` and `model_context_window`, then rounds
the effective usage percentage. This is the default `-context-mode codex`.

## Credits

The HUD concept originates from
[jarrodwatts/claude-hud](https://github.com/jarrodwatts/claude-hud) for Claude
Code. `codex-vitals` is an independent Go implementation for Codex and shares
no code.
