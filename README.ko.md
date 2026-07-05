<div align="center">

# 🫀 codex-vitals

**OpenAI Codex CLI session을 위한 가벼운 Go 패널**
모델, reasoning effort, git 상태, 현재 경로, context 사용량, 5시간/7일 usage limit을 한눈에 보여줍니다.

[English](./README.md) · [한국어](./README.ko.md)

[빠른 설치](#-빠른-설치) · [사이즈](#-사이즈) · [표시 항목](#-표시-항목) · [옵션](#-옵션) · [FAQ](#-faq)

<img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.22+"/>
<img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-89b4fa?style=flat-square" alt="macOS Linux"/>
<img src="https://img.shields.io/badge/output-ANSI%20truecolor-f5c2e7?style=flat-square" alt="ANSI truecolor"/>
<img src="https://img.shields.io/badge/license-MIT-fab387?style=flat-square" alt="MIT"/>

<br/>
<img src="./assets/preview.svg" alt="codex-vitals 패널 미리보기" width="820"/>

</div>

---

## ✨ Highlights

- 🤖 **현재 모델/effort 표시**: `gpt-5.5 ⚡xhigh`처럼 Codex session의 최신 turn 정보를 보여줍니다.
- 🧠 **정확한 context% 계산**: Codex rollout의 token usage를 읽어 Codex 상태 표시와 일치하는 사용률을 계산합니다 ([context% 계산](#-context-계산-기준) 참고).
- 📊 **5H / 7D usage bar**: 5시간, 7일 rate limit 사용률과 reset 시간을 함께 보여줍니다.
- 🌿 **git 상태 표시**: 현재 branch, dirty count, clean 상태를 표시합니다.
- 🎨 **터미널 색상 지원**: ANSI truecolor와 gradient block bar를 사용합니다.
- 🪶 **작은 단일 바이너리**: Go로 작성되어 Node runtime 없이 실행할 수 있습니다.

---

## 🚀 빠른 설치

### 원라이너 (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/WinningBean/codex-vitals/main/install.sh | bash
```

OS/아키텍처에 맞는 prebuilt 바이너리를 `~/.local/bin`에 설치합니다 (릴리스가 아직 없으면 소스 빌드로 폴백).

설치와 동시에 기본 사이즈(`xs`, `s`, `m`, `l`, `xl`)를 지정할 수 있습니다:

```bash
curl -fsSL https://raw.githubusercontent.com/WinningBean/codex-vitals/main/install.sh | bash -s -- xl
```

`~/.config/codex-vitals/size`에 사이즈를 기록합니다. 패널이 이 파일을 매초 다시 읽어서, **떠 있는 패널이 즉시(재시작·셸 리로드 없이) 그 사이즈로 바뀝니다.** 다른 사이즈로 다시 실행하면 언제든 변경됩니다. `-size` 플래그는 단발성으로 이를 덮어씁니다.

### `go install`로 설치

Go 1.22 이상이 있으면 가장 간단합니다.

```bash
go install github.com/WinningBean/codex-vitals/cmd/codex-vitals@latest
```

설치 후 `PATH`에 `$GOPATH/bin` 또는 `$HOME/go/bin`이 잡혀 있어야 합니다.

```bash
codex-vitals -once -context-mode current-hud -size l
```

### 소스에서 직접 빌드

```bash
git clone https://github.com/WinningBean/codex-vitals.git
cd codex-vitals
CGO_ENABLED=0 go build -o codex-vitals ./cmd/codex-vitals
./codex-vitals -once -context-mode current-hud -size l
```

### 로컬 개발 중 실행

```bash
go run ./cmd/codex-vitals -once
go run ./cmd/codex-vitals -once -context-mode current-hud -size l
```

---

## 📐 사이즈

`-size`(기본 `m`)로 패널이 보여주는 정보량을 고릅니다.

| 사이즈 | 줄 | 표시 |
|--------|:--:|------|
| `xs` | 2 | 모델 · 경로 · branch · 미니 바 3개 |
| `s`  | 2 | + 라벨과 퍼센트 |
| `m` (기본) | 4 | 모델 · git · env · 풀폭 context 바 |
| `l`  | 5 | + tokens, session 시간, reset 시각, 20칸 바 |
| `xl` | 5 | 전부, 40칸 바 |

터미널에서는 색상이 적용됩니다. GitHub에서는 아래 텍스트 블록의 색상이 안 보일 수 있습니다.

**`xs`**

```text
🤖 gpt-5.5 ⚡xhigh 📂 ~/Documents/Github/codex-vitals 🌿(main)
🧠 █████░░░░░  🚀 5H ███████░░░  📅 7D ████░░░░░░
```

**`s`**

```text
🤖 gpt-5.5 ⚡xhigh │ 📂 ~/Documents/Github/codex-vitals 🌿(main)
🧠 Context █████░░░░░ 49% │ 🚀 5H ███████░░░ 67% │ 📅 7D ████░░░░░░ 42%
```

**`m` (기본)**

```text
🤖 gpt-5.5 ⚡xhigh │ ✅ clean │ no env
📂 ~/Documents/Github/codex-vitals 🌿(main)
🧠 Context  ████████████████░░░░░░░░░░░░░░░░░ 49% used
🚀 Usage 5H ███████░░░ 67% │ 📅 7D ████░░░░░░ 42%
```

**`l`**

```text
🤖 gpt-5.5 ⚡xhigh │ ✅ clean │ no env
📂 ~/Documents/Github/codex-vitals 🌿(main) │ 🧾 2.1M tokens │ ⏰ 42m
🧠 Context  ██████████░░░░░░░░░░ 49% used (126k/258k)
🚀 Usage 5H █████████████░░░░░░░ 67% (Reset 2h33m left)
📅 Usage 7D ████████░░░░░░░░░░░░ 42% (Reset Fri 16:45)
```

**`xl`**

```text
🤖 gpt-5.5 ⚡xhigh │ ✅ clean │ no env
📂 ~/Documents/Github/codex-vitals 🌿(main) │ 🧾 2.1M tokens │ ⏰ 42m
🧠 Context  ████████████████████░░░░░░░░░░░░░░░░░░░░ 49% used (126k/258k)
🚀 Usage 5H ███████████████████████████░░░░░░░░░░░░░ 67% (Reset 2h33m left)
📅 Usage 7D █████████████████░░░░░░░░░░░░░░░░░░░░░░░ 42% (Reset Fri 16:45)
```

```bash
codex-vitals -once -size xs   # 또는 s, m, l, xl
```

---

## 🎨 표시 항목

| 항목 | 의미 |
|------|------|
| 🤖 **Model** | 현재 Codex session의 최신 모델 |
| ⚡ **Effort** | reasoning effort 값 (`low`, `medium`, `high`, `xhigh` 등) |
| 📝 **Git dirty** | 추적 중인 변경 파일 수 또는 clean 상태 |
| 🐍 **Env** | 활성화된 `CONDA_DEFAULT_ENV` 또는 `VIRTUAL_ENV` |
| 📂 **Path** | 현재 Codex session의 working directory |
| 🌿 **Branch** | 현재 git branch |
| 🧾 **Tokens** | session 누적 token 사용량 |
| ⏰ **Time** | session 시작부터 현재까지의 경과 시간 |
| 🧠 **Context** | context-window 사용률과 token count |
| 🚀 **Usage 5H** | 5시간 usage limit 사용률과 reset까지 남은 시간 |
| 📅 **Usage 7D** | 7일 usage limit 사용률과 reset 시각 |

---

## 🧠 context% 계산 기준

`codex-vitals`는 stock Codex와 patched Codex build를 모두 맞출 수 있도록 두 가지 계산 모드를 제공합니다.

| 모드 | 공식 | 맞추는 대상 |
|------|------|-------------|
| `codex` 기본값 | `(total_tokens - 12000) / (model_context_window - 12000)` | stock Codex 상태 표시 |
| `current-hud` | `(input_tokens + cached_input_tokens) / model_context_window` | 현재 patched Codex build |

stock Codex는 system prompt, tools, `/compact` 여유 공간을 위해 baseline `12000` tokens를 예약합니다. 그래서 기본 `codex` 모드는 `total_tokens`와 `model_context_window`에서 각각 `12000`을 차감합니다. window가 `12000` 이하인 작은 모델에서는 raw 비율로 폴백해, 사용률이 `0%`나 `100%`로 잘못 표시되지 않습니다.

숫자가 Codex 표시와 맞지 않는다면 patched Codex build 공식을 쓰고 있을 가능성이 큽니다. 이 경우 `current-hud`를 사용하세요.

```bash
codex-vitals -once -context-mode current-hud -size l
```

---

## ⚙️ 옵션

```text
-codex-home string
    CODEX_HOME 경로. 기본값은 $CODEX_HOME 또는 ~/.codex

-rollout string
    특정 rollout JSONL 파일을 직접 지정

-context-mode string
    context 사용률 계산 방식: codex 또는 current-hud

-size string
    패널 사이즈: xs, s, m, l, xl
    (기본값: ~/.config/codex-vitals/size, 그다음 CODEX_VITALS_SIZE, 그다음 m)

-interval duration
    반복 출력 주기. 기본값은 1s

-once
    한 번만 렌더링하고 종료

-no-color
    ANSI 색상 없이 출력
```

기본 사이즈는 `~/.config/codex-vitals/size`에 저장됩니다 (설치 스크립트로 기록 — [빠른 설치](#-빠른-설치) 참고 — 또는 `echo l > ~/.config/codex-vitals/size`). 패널이 매초 다시 읽어 실행 중에도 즉시 반영됩니다. `CODEX_VITALS_SIZE`는 폴백이며, `-size` 플래그가 항상 최우선입니다.

---

## 🧪 예시 session으로 확인하기

특정 rollout 파일이 있다면 직접 지정할 수 있습니다.

```bash
codex-vitals \
  -once \
  -rollout /path/to/session.jsonl \
  -context-mode current-hud \
  -size l
```

색상이 깨져 보이는 환경에서는 `--no-color`를 붙이세요.

---

## 📁 읽는 데이터

기본적으로 아래 데이터를 읽습니다.

- `~/.codex/sessions/**/*.jsonl`
- `$CODEX_HOME/config.toml` / `~/.codex/config.toml`
- session rollout의 `session_meta`, `turn_context`, `token_count`
- rate limit의 `primary`, `secondary`

가장 최근 Codex session을 찾아 렌더링하며, `-rollout`으로 특정 파일을 고정할 수 있습니다.

---

## 🧩 tmux / footer에서 쓰기

Codex가 실행 중인 tmux 세션 안에서 아래를 실행하면 하단에 패널이 붙습니다.

```bash
scripts/tmux-panel.sh                 # 사이즈 m, 1초마다 갱신
scripts/tmux-panel.sh -size l
scripts/tmux-panel.sh -size xs
CODEX_VITALS_TMUX_HEIGHT=8 scripts/tmux-panel.sh -size xl
```

패널 높이는 사이즈에 맞춰 자동 조정되며, `CODEX_VITALS_TMUX_HEIGHT`로 덮어쓸 수 있습니다. 포커스는 원래 패널로 돌아오고, 패널에서 Ctrl+C를 누르면 닫힙니다. 직접 실행할 수도 있습니다.

```bash
# 한 번만 출력
codex-vitals -once -context-mode current-hud -size l

# 1초마다 갱신
codex-vitals -context-mode current-hud -size l -interval 1s

# 색상 없이 로그/README용으로 출력
codex-vitals -once -context-mode current-hud -size l --no-color
```

---

## ✅ 요구사항

| 항목 | 필요 이유 |
|------|-----------|
| Go 1.22+ | 빌드 및 `go install` |
| Codex CLI session | `~/.codex/sessions` rollout 데이터 |
| git | branch/dirty 상태 표시 |
| truecolor terminal | gradient bar 색상 표시 |

> `--no-color`를 쓰면 truecolor가 없어도 텍스트 출력은 정상 동작합니다.

---

## 🧹 제거

`go install`로 설치했다면 바이너리만 제거하면 됩니다.

```bash
rm "$(go env GOPATH)/bin/codex-vitals"
```

소스에서 직접 빌드했다면 만든 바이너리만 삭제하세요.

```bash
rm ./codex-vitals
```

`codex-vitals`는 현재 별도의 daemon, launch agent, shell rc 변경을 설치하지 않습니다.

---

## 🙋 FAQ

**색상이 README나 채팅에서 안 보입니다.**
Markdown/채팅 렌더러는 ANSI escape를 색상으로 해석하지 않는 경우가 많습니다. 실제 터미널에서 `--no-color` 없이 실행하면 색상이 보입니다.

**Codex 표시와 숫자가 조금 다릅니다.**
실시간으로 rollout이 계속 갱신되기 때문에 실행 시점 차이로 token count가 1k 단위로 흔들릴 수 있습니다. Codex 표시와 공식을 맞추려면 `-context-mode current-hud`를 사용하세요.

**5H가 left가 아니라 used로 보입니다.**
현재 Codex 표시와 맞추기 위해 usage limit을 사용률(얼마나 썼는지) 기준으로 보여줍니다. reset 시각은 `l`, `xl` 사이즈의 바 옆에 함께 표시됩니다.

**Node가 필요한가요?**
아니요. `codex-vitals` 자체는 Go 바이너리입니다.

**Nerd Font가 필요한가요?**
아니요. 표준 이모지와 Unicode block character를 사용합니다.

---

## Acknowledgements

- 상주형 터미널 패널이라는 아이디어의 원조는 [jarrodwatts/claude-hud](https://github.com/jarrodwatts/claude-hud) (Claude Code용)입니다.
- 레이아웃과 gradient bar는 [AwesomeJun/CC-statusline](https://github.com/AwesomeJun/CC-statusline)에서 영감을 받았습니다.
- [openai/codex](https://github.com/openai/codex)가 기록하는 session 데이터를 읽습니다.

---

<div align="center">

Built for Codex users who want to see the real session vitals at a glance. 🫀

</div>
