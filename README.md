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

For a multi-line terminal footer with emojis and a context bar:

```sh
go run ./cmd/codex-vitals -once -style answer-footer
```

Running a patched Codex build and the context % doesn't match? Add
`-context-mode current-hud` (see [Context percentage](#context-percentage--matching-your-codex)):

```sh
go run ./cmd/codex-vitals -once -style answer-footer -context-mode current-hud
```

By default it reads:

- rollout JSONL files under `~/.codex/sessions`
- config from `$CODEX_HOME/config.toml` or `~/.codex/config.toml`

## Context percentage & matching your Codex

Codex reserves `12000` baseline tokens (system prompt, tools, and room to run
`/compact`). The default `-context-mode codex` mirrors **stock Codex**: it
subtracts the baseline from both `total_tokens` and `model_context_window`, so
the percentage matches what Codex's own status line shows.

```
codex (default):  (total_tokens − 12000) / (context_window − 12000)
current-hud:      (input_tokens + cached_input_tokens) / context_window
```

If codex-vitals' context % does **not** match your Codex status line, you are
probably running a **patched Codex build** whose TUI uses a different formula.
Switch to `-context-mode current-hud` to match it:

```sh
go run ./cmd/codex-vitals -once -context-mode current-hud
```

| Mode | Formula | Matches |
|------|---------|---------|
| `codex` (default) | `(total − 12k) / (window − 12k)` | stock Codex |
| `current-hud` | `(input + cached) / window` | patched forks using that basis |

## Credits

The HUD concept originates from
[jarrodwatts/claude-hud](https://github.com/jarrodwatts/claude-hud) for Claude
Code. `codex-vitals` is an independent Go implementation for Codex and shares
no code.
