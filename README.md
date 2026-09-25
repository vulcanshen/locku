# locku

<p align="center"><img src="docs/icon.svg" width="128" alt="locku icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/locku)](https://github.com/vulcanshen/locku/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/locku)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**Language**: English · [繁體中文](README-zh_TW.md)

**A screensaver with a PIN lock, for the terminal.** locku takes the whole terminal: an LED dot-matrix board spelling the clock, Chrome's offline dinosaur running for ever, or a program of your own. Any key only opens the PIN prompt, and the terminal comes back once the PIN checks out. Wired into tmux as its `lock-command` and into screen as its `LOCKPRG`, or run by hand on a bare tty, so a key after the prefix, or a few idle minutes, locks the screen you walked away from. Without a PIN it is a plain screensaver: any key ends it.

> _When in doubt, hit_ **`Space`**.

locku is the fifth member of the `u`-family — [kbu](https://github.com/vulcanshen/kbu) (Kubernetes), [filu](https://github.com/vulcanshen/filu) (filesystem), [sshu](https://github.com/vulcanshen/sshu) (ssh), [webu](https://github.com/vulcanshen/webu) (browser) — and a lock-screen implementation of [this TUI Design Principle](https://github.com/vulcanshen/thoughts/blob/main/tui-design/README.md). The design, every decision dated in place and the rejected approaches with it, is in [`docs/function.md`](docs/function.md), [`docs/ui.md`](docs/ui.md) and [`docs/ux.md`](docs/ux.md); the developer's notes — the decisions in short, what was rejected, the layout of the code, the tests, the release flow — in [`docs/dev-remarks.md`](docs/dev-remarks.md).

## What you see

Three kinds of saver, as many named profiles of each as you like, one of them active:

- **clock** — the terminal as one LED board: every pixel a Nerd Font square, dark in the profile's `bg`, lit in its `fg`. The time is drawn in a right-angled 3 × 7 pixel font (or 3 × 5), the look of a seven-segment display, as `HH MM` or `HH MM SS`, twenty-four hours, no colon; the date under it in one of four forms, or off. The `column` layout stacks `HH` / `MM` / `SS` and makes the digits several times bigger, the date to their left. Three sizes; what does not fit sheds the year, the seconds, then the date, before the size steps down. Only the pixels that change are redrawn, as a shuffled reveal.
- **dino** — Chrome's offline dinosaur game as a screensaver: the ground and the obstacles scroll by, cacti on the grassland or pyramids in the desert, and the T-Rex jumps them by itself, for ever, never dying; no score, no clock. One runner or two, big or small, drawn as large as the terminal allows.
- **custom** — a program of your own as the picture: `cmatrix -b`, say, or anything else that draws, run through `sh -c`. locku does the lock, the PIN and the integration; the program runs on a pty of locku's, its output passed on as it comes, and the keys never reach it. The PIN prompt goes straight over the moving picture, and unlocking ends the program with the lock.

The terminal each clock size needs (columns × rows, `3x7` / `3x5`):

| Content | small | medium | large |
|---|---|---|---|
| `HH MM` on one line | 38 × 10 / 8 | 62 × 17 / 13 | 96 × 24 / 18 |
| `HH MM SS` on one line | 58 × 10 / 8 | 94 × 17 / 13 | 148 × 24 / 18 |
| `HH` / `MM` stacked | 18 × 18 / 14 | 30 × 32 / 24 | 44 × 47 / 35 |
| `HH` / `MM` / `SS` stacked | 18 × 26 / 20 | 30 × 47 / 35 | 44 × 70 / 52 |

## Install

> locku is **macOS / Linux only** (WSL works). There is no Windows build: the lock stands on the tty, the pty, `su` and tmux / screen.

**Homebrew** (macOS / Linux):

```bash
brew install vulcanshen/tap/locku
```

**Install script** (the latest release binary into `~/.local/bin`, or `/usr/local/bin` as root):

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/locku/main/install.sh | sh
```

**From source**:

```bash
git clone https://github.com/vulcanshen/locku.git
cd locku
make build      # → ./locku (CGO_ENABLED=0, static)
make install    # → $GOBIN
```

**A Nerd Font is required**: every pixel of the board is nf-fa-square, and the settings screen is drawn with Nerd Font glyphs too. Without one the board is a screen of boxes.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/vulcanshen/locku/main/uninstall.sh | sh
```

Removes the binary, then asks — never assumes — about the settings directory. Turn `activate` off on the settings screen first, and locku's blocks are gone from your tmux.conf, screenrc and shell rc; the uninstaller does not touch those files.

## Quick start

```bash
locku            # the settings screen: set a PIN, pick a saver, wire up tmux / screen
locku lock       # lock this terminal now
locku pin reset  # a new PIN, after your login password, when the old one is forgotten
locku version    # the version; locku help prints the usage
```

1. `locku`, then **Settings › preference**, `Enter` on the PIN row, type one. This step is optional: with no PIN locku is a screensaver, and any key ends it.
2. **Integration › tmux** (or **screen**): fill in `config file path` (`~/.tmux.conf` is offered), turn `activate` on, confirm. locku's block is in the file, and on the running tmux server at once.
3. In tmux, `prefix :` then `locku` locks every client on the server. An idle session locks by itself after `lock-after-time` seconds (300 by default). Fill in a `bind-key`, `l` say, and `prefix l` locks too.
4. Any key brings up the PIN prompt; the right PIN gives the terminal back.

`locku lock` on a bare tty — an ssh session, say — locks that terminal the same way. Its `-S` / `-t` flags are what the tmux integration writes into lock-command; you never type them. screen runs the lock as `SCREEN-LOCK` with no arguments at all, and locku answers to that name.

## The lock screen

- Any key opens the PIN prompt, and that key is not input. `Enter` submits, `Esc` goes back to the saver, `Backspace` deletes a digit.
- A wrong PIN turns the border red for a second and swallows every key. `wrong_pin_attempts` wrong in a row start a countdown of `wrong_pin_attempt_cooldown` seconds (off by default); `pin_prompt_timeout` seconds without a key close the prompt (30 by default).
- The status row reads `user@host · locked since HH:MM` (`show_status: false` hides it), and says `no PIN · any key unlocks` when there is none.
- Ctrl+C, Ctrl+Z and Ctrl+\ are just keys; SIGINT / SIGTERM / SIGHUP are ignored; after a panic the lock comes back up. The process ends in three cases only: the right PIN, any key with no PIN set, the terminal going away.
- No PIN set, or a config file that cannot be read, fails open: the saver shows, the status row says why, any key ends it.

**Not a security boundary.** locku keeps stray keys and passers-by off your screen; it does not keep out another session of your own account. What it cannot catch: ssh's `~.` (handled on the client side, the byte never crosses the wire), Linux VT switching (vlock's territory, needs root), `tmux attach -d` / `kill -9` from another SSH session, and the terminal emulator's own shortcuts.

## Forgot the PIN

From any shell of your own:

```bash
locku pin reset
```

It asks `[y/N]`, then your **login password** — your account is the one boundary locku has — and makes a new eight-digit PIN, writes it over the old one and shows it once. A lock already up takes the new PIN at its next key. Change it to one of your own on the settings screen afterwards. Every reset, done or refused, is logged under `~/.locku/data`, without the PIN.

## The settings screen

```
╔[1] locku═════════════╗╭[2] clock  unsaved─────────────────────────────╮
║ Profiles               ║│ Property          Value                          │
║ ● clock                ║│ name              clock                          │
║   clock2               ║│ saver             clock                          │
║   dino                 ║│ layout            row                            │
║ Savers                 ║│ size              medium                         │
║   clock                ║│ time              HH MM                          │
║   dino                 ║│ date              off                            │
║ Integration            ║│ bg                ■ #313244  →  ■ #ff3244        │
║   tmux                 ║│   R               ───────────● 255               │
║   screen               ║│   G               ──●───────── 50                │
║ Settings               ║│   B               ───●──────── 68                │
║   preference           ║│ fg                ■ #f2b753                      │
╚════════════════════════╝╰──────────────────────────────────────────────────╯
 space menu   ? help   tab/1-2 panels   q quit
```

Two panels: **`[1]`** the sidebar, **`[2]`** what the row under the cursor holds, as a Property / Value table. `Tab`, `1` and `2` move between them; `Enter` goes into `[2]` or edits a row; `Esc` closes a float; `Space` lists what can be done here; `?` is the whole key vocabulary, or, on preference's, tmux's or screen's `[2]`, what each row means.

- **Profiles** — the savers you have set up and named. `●` marks the active one, the one the lock shows; `a` makes the row under the cursor active, `p` previews it, `D` duplicates, `r` renames, `X` deletes. Its `[2]` is its settings: clock's `layout`, `size`, `font`, `time`, `date`; dino's `runner`, `scene`; custom's `command`; and `bg` / `fg` as three RGB sliders each, a draft until `S` saves it (`R` drops it, and `q` asks first while one is unsaved). Everything else is written the moment it changes.
- **Savers** — the three kinds: clock, dino, custom. Each `[2]` is a description and the **defaults** a new profile of that kind starts with; `n` makes one, `p` previews the defaults. Changing the defaults touches no existing profile.
- **Integration** — tmux and screen, below.
- **Settings › preference** — the PIN (set it; once set, `Enter` asks the current one and offers `New PIN` or `Remove PIN`), the active `profile`, `show_status`, `pin_prompt_timeout`, `wrong_pin_attempts`, `wrong_pin_attempt_cooldown`.

`P` on any `[2]` previews the lock in place: a profile's or a saver's `[2]` shows that one, any other the active profile. Any key comes back, and no PIN is asked. A custom profile's preview hands the terminal to the program until a key.

## tmux and screen

Integration › tmux and Integration › screen each have `activate` (on / off), `config file path` (the file to write; `~/.tmux.conf` and `~/.screenrc` are offered) and, under a rule, the tool's own keys:

| | tmux | screen |
|---|---|---|
| Idle seconds before the tool locks by itself (0 never) | `lock-after-time` | `idle` |
| The key after the prefix that locks, in the tool's own spelling; empty binds nothing | `bind-key` (`l`, `C-l`) | `bind` (`l`, `^L`); `C-a x` locks anyway |
| What locks | `lock`: `lock-server` (every client on the server) or `lock-session` (this session only) | — (a screen has no server to scope) |

`activate` on writes locku's block into the file, after a confirm, and while it is on rewrites it the moment any row changes: a running tmux server takes the whole block at once, and running screens take `idle` and `bind` at once. Off takes the block out again, from the file and from what is running. Only the block between the markers is ever touched, every line of it ending in `# locku`; the rest of the file is yours. With no path filled in, `activate` cannot be pressed: nothing is guessed.

What tmux gets:

```
# >>> locku >>>
set -gF lock-command "/opt/homebrew/bin/locku lock -S '#{socket_path}'"  # locku
set -g lock-after-time 300                                                  # locku: 0 never
set -s "command-alias[90]" "locku=lock-server"                              # locku: prefix : locku locks every client
set-hook -g "client-attached[90]" "if -F \"#{@locked}\" lock-client"        # locku: attaching while locked locks the client
set-hook -g "client-session-changed[90]" "if -F \"#{@locked}\" lock-client" # locku: so does switching sessions
bind-key l lock-server                                                      # locku: prefix l locks every client
# <<< locku <<<
```

`prefix :` then `locku` locks, and so does `prefix l` when `bind-key` is `l`. The server stays locked for whoever comes: attaching to it, or switching sessions, while it is locked lands on the screensaver too, until a PIN unlocks it. With `lock` set to `lock-session` the same lines point at `lock-session`, one more hook gives every session its own lock-command, and the other sessions carry on.

What screen gets:

```
# >>> locku >>>
idle 300 lockscreen   # locku: 0 never
bind l lockscreen     # locku: C-a l locks, as C-a x does
# <<< locku <<<
```

and, because screen reads its lock program from the shell that started it and never from the screenrc, the shell rc (`~/.zshrc`, `~/.bashrc`, or fish's `config.fish`) gets:

```
# >>> locku >>>
export LOCKPRG=/usr/local/bin/locku   # locku: screen's LOCKPRG
# <<< locku <<<
```

A new shell has it. A screen already running gets it once detached and attached again from a new shell; until then that session locks with screen's own built-in lock, which the settings screen says. The blocks are the same text whether locku writes them or you do.

## Where your data lives

| | What | Where |
|---|---|---|
| settings | `config.yaml` — the PIN's hash, the profiles, the savers' defaults, the integration | `~/.config/locku` (`$XDG_CONFIG_HOME/locku` when set; `$LOCKU_CONFIG` names it outright) |
| data | `pin-resets.log` — every `locku pin reset`, without the PIN | `~/.locku/data` (`$LOCKU_DATA`) |

No history, no cache, no session. `config.yaml` is written atomically, mode 0600, and can be edited by hand:

```yaml
auth: pin                 # the only check there is; pam is reserved
pin_hash: "$2a$10$..."    # bcrypt; empty or missing = no PIN, any key unlocks
profile: clock            # the active profile
profiles:
  - name: clock
    saver: clock          # clock / dino / custom; fixed once made
    layout: row           # row / column
    size: medium          # small / medium / large: one pixel is 1 / 2 / 3 cells square
    font: 3x7             # 3x7 / 3x5
    time: "HH MM"         # HH MM / HH MM SS
    date: off             # off / YYYY-MM-DD / YYYY-MMM-DD / MM-DD / MMM-DD
    bg: "#313244"
    fg: "#f2b753"
  - name: dino
    saver: dino
    runner: big           # big / small / big-big / small-small / small-big / big-small
    scene: grassland      # grassland / desert
    bg: "#313244"
    fg: "#f2b753"
  - name: matrix
    saver: custom
    command: "cmatrix -b" # run through sh -c; no colours of its own
savers:                   # each kind's defaults: what a new profile starts as
  clock: { saver: clock, layout: row, size: large, font: 3x5, time: "HH MM SS", date: YYYY-MM-DD, bg: "#313244", fg: "#f2b753" }
  dino: { saver: dino, runner: big, scene: grassland, bg: "#313244", fg: "#f2b753" }
  custom: { saver: custom, command: "" }
show_status: true               # the user@host · locked since row
pin_prompt_timeout: 30          # seconds without a key before the prompt closes; 0 never
wrong_pin_attempts: 0           # wrong PINs in a row before a cooldown; 0 off
wrong_pin_attempt_cooldown: 30  # the cooldown, in seconds
tmux:
  conf: "~/.tmux.conf"          # config file path; empty = activate cannot be turned on
  lock-after-time: 300
  bind-key: ""
  lock: lock-server             # lock-server / lock-session
screen:
  conf: "~/.screenrc"
  idle: 300
  bind: ""
```

## Key bindings

### Everywhere

| Key | |
|---|---|
| `Tab` · `1` · `2` | next panel / this panel |
| `Enter` | on `[1]`: the row's fields, in `[2]`; on `[2]`: edit, choose, toggle, pick |
| `Esc` | close the top float |
| `Space` | what can I do here: the item, and the panel |
| `?` | help: the keys; on preference's, tmux's or screen's `[2]`, what each row means |
| `P` | on `[2]`: preview the lock; any key comes back |
| `q` | quit; asks first when colours are unsaved |
| `j` / `k` · `u` / `d` · `gg` / `G` | down / up · half a page · first / last |

### `[1]` sidebar

| Key | On | |
|---|---|---|
| `n` | a saver | new profile of this kind, under a name |
| `p` | a saver / a profile | preview the defaults / this profile |
| `a` | a profile | activate: the lock shows this profile from now on |
| `D` · `r` · `X` | a profile | duplicate · rename · delete (not the active one, not the last one) |

### `[2]` detail

| Key | On | |
|---|---|---|
| `Enter` | any row | rename, choose, toggle, pick a colour channel, set or change the PIN, edit a path or a key, turn `activate` |
| `S` · `R` | a profile or a saver | save the colour draft · drop it |

### The lock screen

| Key | |
|---|---|
| any key | open the PIN prompt (the key is not input) |
| `Enter` · `Esc` · `Backspace` | submit · back to the saver · delete a digit |

## Status

**v0.1.0** — the three savers, the PIN and `locku pin reset`, tmux and screen integration with `activate`. See [CHANGELOG.md](CHANGELOG.md).

Not there, on purpose:
- **Windows** — the lock stands on the tty, the pty, `su` and tmux / screen; a native port would be another product
- **the system password** — the only check is locku's own PIN; `auth: pam` is reserved, nothing more
- **locking the Linux virtual console** (Alt+F1 … F7) — vlock's territory
- **a "forgot my PIN" entry on the lock screen** — the way back is `locku pin reset`, from a shell, with your login password

## Built with

Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss), [bubbletea-overlay](https://github.com/rmhubbert/bubbletea-overlay) for the floats, [creack/pty](https://github.com/creack/pty) for the custom saver's program and the login check, `golang.org/x/crypto/bcrypt` for the PIN, and `gopkg.in/yaml.v3` for the config. Colours are catppuccin-mocha.

## Docs

| File | Answers | Read |
|---|---|---|
| [`docs/function.md`](docs/function.md) | Why the lock stands outside tmux / screen, the one contract for the three entry points, the signal table, the state machine, the PIN and the no-PIN mode, the three savers, the canvas renderer, the CLI, the config, how Integration writes the files, the decision list, the acceptance | 1st |
| [`docs/ui.md`](docs/ui.md) | The grid of the settings screen's two panels, the lock canvas, how every field is drawn, the popups, the PIN prompt's four states, the palette, saving | 2nd |
| [`docs/ux.md`](docs/ux.md) | Core-key semantics, what the Space menu holds, the global `?`, how each field is filled in, the PIN's three questions, the hotkey layers, the floats, the timeline | 3rd |
| [`docs/dev-remarks.md`](docs/dev-remarks.md) | The developer's notes: where things stand, the decisions in short, what was rejected, the layout of the code, the tests, the release flow, what is next | — |
| [`docs/icon.svg`](docs/icon.svg) | The icon: the family's block-letter mark on a black square | — |

The docs are in Traditional Chinese, every decision dated in place.

## Development

```
make build     → ./locku
make check     fmt-check + vet + go test -race, before a commit
make e2e       end to end on a real tmux, a pty and a real screen (needs tmux, screen and python3); your own servers and sessions are not touched
make lock      build and lock this terminal
```

What the tests cover, and how a release is cut: [`docs/dev-remarks.md`](docs/dev-remarks.md).

## License

[GPL-3.0](LICENSE)
