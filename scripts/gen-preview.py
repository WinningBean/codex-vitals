#!/usr/bin/env python3
# Generate assets/preview.svg: one terminal box per -size preset (xs..xl),
# stacked vertically, CC-statusline style. Bars mirror render.go gradientColor.
import math

def around(x):
    return math.floor(x + 0.5) if x >= 0 else math.ceil(x - 0.5)

def lerp(a, b, t):
    return around(a + (b - a) * t / 100)

def gradient(percent, kind):
    percent = max(0, min(100, around(percent)))
    if kind == "context":
        if percent < 30:
            t = percent * 100 / 30
            return (lerp(245, 230, t), lerp(194, 69, t), lerp(231, 83, t))
        if percent < 70:
            t = (percent - 30) * 100 / 40
            return (lerp(230, 210, t), lerp(69, 15, t), lerp(83, 57, t))
        return (210, 15, 57)
    if kind == "7d":
        if percent < 50:
            t = percent * 2
            return (lerp(249, 254, t), lerp(226, 100, t), lerp(175, 11, t))
        t = (percent - 50) * 2
        return (lerp(254, 210, t), lerp(100, 15, t), lerp(11, 57, t))
    if percent < 50:
        t = percent * 2
        return (lerp(180, 30, t), lerp(190, 102, t), lerp(254, 245, t))
    t = (percent - 50) * 2
    return (lerp(30, 210, t), lerp(102, 15, t), lerp(245, 57, t))

def hexc(rgb):
    return "#%02x%02x%02x" % rgb

DIM = "#585b70"
TEAL = "#94e2d5"
PATH = "#a6adc8"
GREEN = "#40a02b"
PEACH = "#fab387"
PINK = "#f5c2e7"
BLUE = "#b4befe"
YELLOW = "#f9e2af"

CTX, H5, D7 = 49, 67, 42

def bar_segments(percent, width, kind):
    filled = max(0, min(width, around(percent / 100 * width)))
    segs = [("█", hexc(gradient(i * 100 / width, kind))) for i in range(filled)]
    if width - filled > 0:
        segs.append(("░" * (width - filled), hexc(gradient(percent, kind))))
    return segs

def pct(percent, kind, extra=""):
    txt = "%d%%" % percent + (" " + extra if extra else "")
    return [(txt, hexc(gradient(percent, kind)))]

def size_xs():
    return [
        [("🤖 ", DIM), ("gpt-5.5 ⚡xhigh", TEAL), (" 📂 ", DIM),
         ("~/Documents/Github/codex-vitals", PATH), (" ", DIM), ("🌿(main)", GREEN)],
        [("🧠 ", DIM)] + bar_segments(CTX, 10, "context") +
        [("  🚀 5H ", DIM)] + bar_segments(H5, 10, "default") +
        [("  📅 7D ", DIM)] + bar_segments(D7, 10, "7d"),
    ]

def size_s():
    return [
        [("🤖 ", DIM), ("gpt-5.5 ⚡xhigh", TEAL), (" │ ", DIM), ("📂 ", DIM),
         ("~/Documents/Github/codex-vitals", PATH), (" ", DIM), ("🌿(main)", GREEN)],
        [("🧠 ", DIM), ("Context", PINK), (" ", DIM)] + bar_segments(CTX, 10, "context") +
        [(" ", DIM)] + pct(CTX, "context") + [(" │ ", DIM), ("🚀 ", DIM), ("5H", BLUE), (" ", DIM)] +
        bar_segments(H5, 10, "default") + [(" ", DIM)] + pct(H5, "default") +
        [(" │ ", DIM), ("📅 ", DIM), ("7D", YELLOW), (" ", DIM)] + bar_segments(D7, 10, "7d") +
        [(" ", DIM)] + pct(D7, "7d"),
    ]

def head_line():
    return [("🤖 ", DIM), ("gpt-5.5 ⚡xhigh", TEAL), (" │ ", DIM),
            ("✅ clean", GREEN), (" │ ", DIM), ("no env", DIM)]

def size_m():
    return [
        head_line(),
        [("📂 ", DIM), ("~/Documents/Github/codex-vitals", PATH), (" ", DIM), ("🌿(main)", GREEN)],
        [("🧠 ", DIM), ("Context ", PINK), (" ", DIM)] + bar_segments(CTX, 33, "context") +
        [(" ", DIM)] + pct(CTX, "context") + [(" ", DIM), ("used", DIM)],
        [("🚀 ", DIM), ("Usage 5H", BLUE), (" ", DIM)] + bar_segments(H5, 10, "default") +
        [(" ", DIM)] + pct(H5, "default") + [(" │ ", DIM), ("📅 ", DIM), ("7D", YELLOW), (" ", DIM)] +
        bar_segments(D7, 10, "7d") + [(" ", DIM)] + pct(D7, "7d"),
    ]

def size_large(width):
    return [
        head_line(),
        [("📂 ", DIM), ("~/Documents/Github/codex-vitals", PATH), (" ", DIM), ("🌿(main)", GREEN),
         (" │ 🧾 ", DIM), ("2.1M tokens", PEACH), (" │ ", DIM), ("⏰ 42m", DIM)],
        [("🧠 ", DIM), ("Context ", PINK), (" ", DIM)] + bar_segments(CTX, width, "context") +
        [(" ", DIM)] + pct(CTX, "context", "used") + [(" ", DIM), ("(126k/258k)", DIM)],
        [("🚀 ", DIM), ("Usage 5H", BLUE), (" ", DIM)] + bar_segments(H5, width, "default") +
        [(" ", DIM)] + pct(H5, "default") + [(" ", DIM), ("(Reset 2h33m left)", DIM)],
        [("📅 ", DIM), ("Usage 7D", YELLOW), (" ", DIM)] + bar_segments(D7, width, "7d") +
        [(" ", DIM)] + pct(D7, "7d") + [(" ", DIM), ("(Reset Fri 16:45)", DIM)],
    ]

sections = [
    ("xs", size_xs()),
    ("s", size_s()),
    ("m", size_m()),
    ("l", size_large(20)),
    ("xl", size_large(40)),
]

def esc(t):
    return t.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

def is_wide(ch):
    o = ord(ch)
    return o >= 0x1F000 or o in (0x26A1, 0x2705, 0x23F0) or 0x2600 <= o <= 0x27BF

def cols(text):
    return sum(2 if is_wide(c) else 1 for c in text)

def line_cols(line):
    return sum(cols(t) for t, _ in line)

# Metrics
PX = 9.2          # px per monospace column at font-size 15
PAD_L = 16
PAD_R = 22
TITLE_H = 34
TOP_PAD = 12
BOT_PAD = 14
LINE_H = 22
CARD_GAP = 18
OUTER = 8         # margin around the whole stack

# Compute each card's width/height
cards = []
for name, lines in sections:
    maxcols = max(line_cols(l) for l in lines)
    w = PAD_L + int(round(maxcols * PX)) + PAD_R
    w = max(w, 300)
    h = TITLE_H + TOP_PAD + LINE_H * len(lines) + BOT_PAD
    cards.append((name, lines, w, h))

W = max(c[2] for c in cards) + OUTER * 2
H = OUTER * 2 + sum(c[3] for c in cards) + CARD_GAP * (len(cards) - 1)

svg = []
svg.append(f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" '
           f'font-family="ui-monospace, SFMono-Regular, Menlo, Consolas, monospace">')
# No canvas background: inter-card gaps stay transparent so the image
# adapts to whatever page (light or dark) embeds it.

y = OUTER
for name, lines, cw, ch in cards:
    x = OUTER
    # card body + title bar
    svg.append(f'<rect x="{x}" y="{y}" width="{cw}" height="{ch}" rx="12" fill="#181825"/>')
    svg.append(f'<rect x="{x}" y="{y}" width="{cw}" height="{TITLE_H}" rx="12" fill="#11111b"/>'
               f'<rect x="{x}" y="{y+TITLE_H-12}" width="{cw}" height="12" fill="#11111b"/>')
    cy = y + 17
    svg.append(f'<circle cx="{x+22}" cy="{cy}" r="6" fill="#f38ba8"/>'
               f'<circle cx="{x+40}" cy="{cy}" r="6" fill="#f9e2af"/>'
               f'<circle cx="{x+58}" cy="{cy}" r="6" fill="#a6e3a1"/>')
    svg.append(f'<text x="{x+cw//2}" y="{y+21}" text-anchor="middle" fill="#6c7086" '
               f'font-size="12">codex-vitals -size {name}</text>')
    # content
    ty = y + TITLE_H + TOP_PAD + 15
    for line in lines:
        parts = "".join(f'<tspan fill="{c}">{esc(t)}</tspan>' for t, c in line)
        svg.append(f'<text x="{x+PAD_L}" y="{ty}" font-size="15" xml:space="preserve">{parts}</text>')
        ty += LINE_H
    y += ch + CARD_GAP

svg.append('</svg>')

with open("assets/preview.svg", "w", encoding="utf-8") as f:
    f.write("\n".join(svg) + "\n")
print(f"wrote assets/preview.svg  ({W}x{H})")
for name, lines, cw, ch in cards:
    print(f"  {name:>2}: {cw}x{ch}")
