# locku

<p align="center"><img src="docs/icon.svg" width="128" alt="locku icon" /></p>

[![GitHub Release](https://img.shields.io/github/v/release/vulcanshen/locku)](https://github.com/vulcanshen/locku/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vulcanshen/locku)](https://go.dev/)
[![License](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

**Language**: English · [繁體中文](README-zh_TW.md)

A screensaver with a PIN lock for the terminal. The fifth member of the `u`-family (kbu / filu / sshu / webu / locku), built to [VTP](https://github.com/vulcanshen/thoughts/blob/main/tui-design/README.md), the family's TUI design principle; the icon is the family's block-letter mark, [`docs/icon.svg`](docs/icon.svg).

Run as tmux's `lock-command`, as screen's `LOCKPRG`, or straight on a bare tty: tmux / screen hand it the real tty,
and it fills the whole screen as an LED dot-matrix board drawing the clock, runs the dinosaur, or runs a program of your own; any key only brings up the PIN prompt, and the tty is handed back once the PIN checks out.
With no PIN set it is a plain screensaver: any key unlocks.

> The design was settled on 2026-09-24 and the first version built the same day; 2026-09-25 added Integration's `activate`, the dino and custom savers and `locku pin reset`, and the docs were rewritten to match the code the same day.
> The unit tests (`make check`, with the race detector) and the three pty end-to-end suites (tmux / custom / screen) all pass. v0.1.0 was released on 2026-09-25: the GitHub Release carries the tarballs for four platforms and their checksums, and the brew formula is in vulcanshen/homebrew-tap.

**Platform**: macOS and Linux (WSL works). Windows is not supported: the lock stands on the tty, the pty, `su` and tmux / screen; a native port is another product (2026-09-25, the user's call).

## Four commands

| Command | Does |
|---|---|
| `locku` | The settings TUI: three savers (clock, dino, custom), each with as many named profiles as you like, every profile's settings and bg / fg colours (custom has only `command`, no colours); preference's PIN, the active profile, show_status, pin_prompt_timeout, wrong_pin_attempts / wrong_pin_attempt_cooldown; Integration's tmux / screen (below). Every change is written to the file at once, except the colours, which are a draft (`S` saves, `R` drops). `P` on `[2]` previews in place (any key comes back; no PIN asked) |
| `locku lock` | Locks the current tty. tmux, screen and a bare tty all call this. `-S <socket>` / `-t <session>` are what the tmux integration writes into lock-command; you never type them |
| `locku pin reset` | The way back from a forgotten PIN, below |
| `locku version` | The version (`locku help` prints the usage) |

When argv[0] is `SCREEN-LOCK` it counts as `locku lock`, because screen's LOCKPRG is an execl and takes no arguments.

## Forgot the PIN

Run `locku pin reset` from any shell of your own: it asks `[y/N]`, then your **login password** (the account is the one boundary; a static binary has no PAM, so the password is checked with `su` on a pty), and only past both does it make a new eight-digit PIN, write its bcrypt over the config's `pin_hash` and show it once. The PIN is never emptied and no lock is left open; change it to one of your own on the settings screen afterwards. A lock already up re-reads `pin_hash` at every key, so the next key knows the new PIN and the old one stops working; a broken file keeps the hash the lock started with, and an emptied `pin_hash` counts as no PIN. Every reset, done or refused for a wrong password, is logged in `~/.locku/data/pin-resets.log` (`$LOCKU_DATA` moves it), without the PIN. It does not run without a terminal, nor when the config cannot be read. Anyone who can do this could kill the lock anyway: locku is not a security boundary, the account is.

## Install

```bash
git clone https://github.com/vulcanshen/locku.git
cd locku
make build      # → ./locku (CGO_ENABLED=0, static)
./locku         # Integration › tmux / screen: fill in config file path, turn activate on to write the integration; set a PIN (optional: with none it is a plain screensaver)
./locku lock    # lock right now
```

Or without the source:

```bash
brew install vulcanshen/tap/locku
# or
curl -fsSL https://raw.githubusercontent.com/vulcanshen/locku/main/install.sh | sh
```

**A Nerd Font is required**: every pixel of the board is nf-fa-square.

## The lock screen

Any key opens the PIN prompt, and that key does not count as input; `Enter` submits, `Esc` goes back to the saver, `Backspace` deletes one digit.
A wrong PIN turns the border red for a second and swallows every key; `wrong_pin_attempts` wrong in a row start a `wrong_pin_attempt_cooldown`-second countdown;
`pin_prompt_timeout` seconds without a key close the prompt. Locking after idle is tmux's `lock-after-time` and screen's `idle` (one each, written as they are while activate is on and rewritten on a change; default 300, 0 off). The status row reads `user@host · locked since HH:MM`, and says `no PIN · any key unlocks` when there is none.

Ctrl+C, Ctrl+Z and Ctrl+\ are just keys; SIGINT / SIGTERM / SIGHUP are ignored; after a panic the lock screen comes back up.
The process ends in three cases only: the right PIN, any key in the no-PIN mode, the tty going away.

What it cannot catch (not a security boundary; `docs/function.md` §0.1, §2.3): ssh's `~.`, Linux VT switching, `tmux attach -d` / `kill -9` from another SSH session, the terminal emulator's own shortcuts.

## Your own program as the saver (custom)

Pick custom for a profile and give it a `command` (run through `sh -c`, `cmatrix -b` say); locku does the lock, the PIN and the integration. The program runs on a pty locku opens, in a process group of its own, its output passed to the terminal as it comes, and the keys stay with locku; locku does not restart it or read its picture, and unlocking SIGKILLs the whole group. A key puts the PIN box straight over the moving picture: after every chunk the program writes, the box is painted again behind it, inside one synchronised update, and no frame is dropped; Esc or the timeout blanks the box's place and the picture goes on, and only a program that has sat still for half a second is asked to repaint. A program that ends (it should not) does not end the lock: locku's own board writes `EXIT <code>` as it is (`EXIT` in gold, the number green for 0 and peach for the rest; killed by a signal is 128 + its number; no command, or one that could not start, is a red `NONE`), and the status row says why in red. custom has no bg / fg: the picture is the program's. The command is not sanitised in any way: it is your own command on your own machine. The preview from the settings screen hands the whole terminal to the program; any key comes back.

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

**Profiles** are the savers you have set up and named (objects): new / duplicate / rename / delete live here, `●` marks the active one, and `a` makes the one under the cursor active (2026-09-25).
**Savers** are the kinds (classes): clock, dino, custom, unnamed and fixed in number; their `[2]` is a description plus the **defaults**, which is how a profile made from that saver starts out afterwards,
and changing them touches no existing profile; `n` on one makes a profile of it, `p` previews the defaults. The built-in defaults: clock is row / large / 3x5 / `HH MM SS` / `YYYY-MM-DD`, dino is big / grassland, custom's command is empty.

`Tab` / `1` / `2` switch panels, `Enter` goes into `[2]` or edits, `Esc` closes a float, `Space` lists what can be done here, `?` the global actions.
A saver in `[1]`: `n` new profile, `p` preview the defaults; a profile in `[1]`: `p` preview this profile, `a` activate it, `D` duplicate, `r` rename, `X` delete; on a profile's or saver's `[2]`: `P` preview, `S` save the colour draft, `R` drop it (custom has only `P`);
the PIN row of preference's `[2]`: with a PIN set, Enter checks the current one first, then offers `New PIN` or `Remove PIN` (Remove takes effect at once). Which profile is active is chosen at preference › profile, or with `a` on the profile's row in the sidebar; `●` marks it. `P` previews only on `[2]` (a profile's or saver's `[2]` shows that one, any other the active one; on `[1]` it does nothing, `p` is the row under the cursor), `q` quits.
**Integration**'s tmux / screen `[2]` is `activate` (on / off), `config file path` (the file to write), a rule, then the tool's own keys: tmux's `lock` (`lock-server` for the whole server, or `lock-session` for this session only), the idle lock under the tool's own name, `lock-after-time` for tmux and `idle` for screen, tmux's `bind-key` (the key after the prefix that locks, in tmux's spelling, `l`, `C-l`, written as `bind-key <key> <lock>`) and screen's `bind` (the key after C-a that locks, in screen's spelling, `l`, `^L`, written as `bind <key> lockscreen`; `C-a x` locks anyway, built in), and an empty one binds nothing; screen has no `lock`, having no server to scope. `activate` is whether the block is in the file, written after a confirm on Enter; while it is on, changing any row rewrites the block at once, a running tmux server gets the whole block by `source-file` and screen's running sessions get it by `screen -X` at once; turning it off takes the block out. Only the `# >>> locku >>>` … `# <<< locku <<<` block is ever touched, every line of it ending in `# locku`, idempotently; with no path filled in the row cannot be pressed, and nothing is guessed. The first row of every `[2]` is the `Property` / `Value` header; the title is the family's powerline capsule: `[2]` the name (with `unsaved` after it while a draft is unsaved), `[1] locku`. `config file path` is one of the only two free-text inputs, in the manner webu proposed: the box shows the current value dimmed (or `~/.tmux.conf` / `~/.screenrc` when there is none), `Tab` takes it over for editing, `Backspace` refuses it, and Enter untouched changes nothing. What each setting means: `?` with the focus on preference's or tmux's / screen's `[2]` lists only that panel's settings (wrapped); anywhere else `?` is the keys.

## Docs

| File | Answers | Read |
|---|---|---|
| [`docs/function.md`](docs/function.md) | Why the prefix cannot be caught inside a pane, the one contract for the three entry points, the signal table, the state machine, PIN checking, the no-PIN mode, `locku pin reset`, the three savers (with custom's pty and the box over it), the canvas renderer and its fallbacks, the CLI, the config, how Integration writes the files (with what was measured in tmux / screen), the technology and platform choices, the 41 decisions, the MVP acceptance | 1st |
| [`docs/ui.md`](docs/ui.md) | The grid of the settings screen's two panels, the lock canvas's board and the `EXIT` board, how every field is drawn, the popup list, the PIN prompt's four states, the palette, the chrome, saving | 2nd |
| [`docs/ux.md`](docs/ux.md) | Core-key semantics, what the Space menu holds, the global `?`, how each field is filled in, the PIN's three questions, the canvas prompt's event table, the hotkey layers and the clash check, the floats, the timeline | 3rd |
| [`docs/icon.svg`](docs/icon.svg) | The icon: the family's block-letter mark on a black square, locku's version (2026-09-25) | — |

All three docs are in Traditional Chinese, every decision dated in place, and every open-questions list is empty.

## Decisions, in short

- **The lock stands outside tmux / screen, not inside a pane.** A program inside a pane never sees the prefix; tmux's lock-command is run synchronously by the client process with `system()`, which reads no key meanwhile, and screen's LOCKPRG is the same. It is the one correct hook.
- **tmux: the whole server locks by default (`lock` can make it this session only, 2026-09-25), no key is bound by default, and whoever comes in while it is locked is locked (2026-09-24).** `locku` typed at `prefix :` is `lock-server` (a command alias, clashing with none of your binds; for a key, fill one in at `bind-key`, 2026-09-25), and every client of every session becomes the screensaver together; tmux has no "locked" state of its own, so locku sets the global `@locked` as `locku lock` starts, and the `client-attached` / `client-session-changed` hooks `lock-client` on seeing it, whichever session is attached; the right PIN clears it, the tty going away does not. The idle lock is tmux's, timed per session: the screen that sits idle is the one that locks. lock-command is locku's absolute path, with `#{socket_path}` expanded by tmux into `locku lock -S`, so a non-default socket is right too. With `lock-session` the flag stands on the session and the hooks stay; the other sessions carry on; the lock learns its session from its own session's lock-command (`-t`), because looking it up from the tty while locked gets the wrong session (measured). A running server gets the whole block (`source-file`, 2026-09-25): the same text as the file, after undoing what the old block did and the new one does not, then each existing session's own lock-command is set.
- **screen: the same idea as tmux, in screen's names (2026-09-25).** `idle N lockscreen` and `bind <key> lockscreen` go into screenrc, and to every running session by `screen -X` at once; LOCKPRG can only come through the shell's environment (measured 2026-09-24: `setenv` in `.screenrc` does nothing for the lock, which the attacher reads with `getenv`), so activate writes the shell rc as well; a new shell has it, and a session already running gets it once detached and re-attached with `screen -r` from a new shell; until then those sessions lock with screen's built-in `Key:` lock, which the toast and `?` both say. No `lock`: screen has no server, so there is no scope to choose.
- **custom saver: your own program as the saver's animation (2026-09-25).** A profile holds one `command` (run through `sh -c`, `cmatrix -b` say); locku does the lock, the PIN and the integration. The program runs on a pty locku opens, in a process group of its own, its output passed straight to the terminal, the keys always with locku; locku does not restart it or read its picture, and unlocking SIGKILLs the whole group. The PIN box sits over the moving picture: painted again after every frame, wrapped in DECSC / DECRC, inside one synchronised update, no frame dropped; closing it blanks its place, and SIGWINCH goes only to a program idle for half a second or more (one that is drawing flashes and starts over when asked to repaint, measured with cmatrix). A program that ends (it should not) does not end the lock: locku's board writes `EXIT <code>` as it is (`EXIT` in gold, the number green for 0 and peach for the rest; killed by a signal is 128 + its number, no command is a red `NONE`), and the status row says why in red. custom has no bg / fg: the picture is the program's. The command is not sanitised.
- **Forgot the PIN: `locku pin reset` (2026-09-25).** After `[y/N]` it asks the login password (checked with `su` on a pty), then makes a new PIN, writes it over the config and shows it once (elasticsearch's reset-password, not an emptied `pin_hash`), and logs it to `~/.locku/data/pin-resets.log` (without the PIN); a lock already up re-reads `pin_hash` at every key.
- **Process alive = locked, ended = unlocked.** No error may end the process; only the right PIN, any key in the no-PIN mode, and the tty going away end it.
- **Not a security boundary.** A second SSH session can kill it. It is a screensaver and a guard against stray keys; a missing or broken config always fails open.
- **The only check is locku's own PIN**, bcrypt in the config; `auth: pam` is left as an extension point, shadow is not done. A wrong PIN is a fixed one-second debounce; a lockout after repeated wrong PINs is configurable, off by default.
- **A saver is a class, a profile an object (settled 2026-09-24).** Three savers: clock, dino, and custom, your own program (2026-09-25); a profile is a named, set-up one, the file's `profile` points at it, and it is what the lock screen shows. A profile is made from a saver with `n`, and its saver does not change afterwards. dino is Chrome's offline dinosaur game as a screensaver: the ground and the cacti scroll left and the T-Rex jumps them by itself, for ever, random obstacles, random jumps, never dying, no score and no clock; its settings are only runner (big / small, one large or small T-Rex; big-big / small-small / small-big / big-small, two one behind the other, the name being their order on the screen left to right, each jumping on its own; settled 2026-09-25, the old trex / two-trex converted), scene (grassland with cacti, or desert with pyramids), bg / fg, and no size: the canvas takes the largest scale that fits; a frame every 70 ms. clock's layout row / column (column splits `HH` / `MM` / `SS` into lines, the digits several times larger), size small / medium / large (one font pixel is 1 / 2 / 3 cells square), font 3x7 / 3x5, time `HH MM` / `HH MM SS` (24-hour, no colon, the groups parted by a space), date off or one of four, and the two colours bg / fg; no free-text input anywhere; duplicate / rename / delete.
- **The canvas is drawn one way only: a whole LED dot-matrix board.** Every cell is nf-fa-square plus a space, a dark cell in the saver's bg, a lit one in its fg. The glyphs look like a seven-segment display: all right angles, no diagonals, no slash through the zero, the digits 3 × 7, scaled by size; letter spacing, line spacing and the space inside the time are gap units of their own (1 cell at small / medium, 2 at large), not scaled with the pixels. The time and the date are two blocks of their own: the time is laid out first, the date takes what is left (below in row, to the left in column), each stepping its size down before dropping a unit (the time its seconds, the date its year); a date that does not fit is not drawn, and only a time that does not fit falls back to plain text. The first frame is not animated; after that only the pixels that changed get a splash-style shuffle. The character set is 39 glyphs.

  The terminal each size needs (columns × rows, 3x7 / 3x5):

  | Content | small | medium | large |
  |---|---|---|---|
  | `HH MM` on one line | 38 × 10 / 8 | 62 × 17 / 13 | 96 × 24 / 18 |
  | `HH MM SS` on one line | 58 × 10 / 8 | 94 × 17 / 13 | 148 × 24 / 18 |
  | `HH` / `MM` stacked | 18 × 18 / 14 | 30 × 32 / 24 | 44 × 47 / 35 |
  | `HH` / `MM` / `SS` stacked | 18 × 26 / 20 | 30 × 47 / 35 | 44 × 70 / 52 |
- **Colours belong to each saver**, three RGB sliders each for bg and fg, in webu's number-list manner, no typing; a slider is drawn in its own channel's colour (the R row in `#RR0000`); it changes a draft, `S` writes the file, `R` drops it, and `q` asks first when a draft is unsaved. custom has no colours.
- **Platform: macOS / Linux (WSL works), no Windows (2026-09-25).** The lock stands on the tty, the pty, `su` and tmux / screen.
- **Enter = into `[2]` / edit / submit, Esc only cancels, `X` deletes, `d` is half a page.** On the canvas any key only opens the prompt, and the first key is not input.

## Rejected, do not bring back

Catching the prefix inside a pane, attaching to the user's existing session, locking up when the config is missing, PAM / shadow in v1, a free-text saver, free strftime formats,
a marquee, dropping the space between pixels, a preview box inside `[2]`, a base sheet, an Integration popup, a `--saver` command-line override,
`locku init`, a print-only setup, a `locku setup` command, the later `S` / `X` hotkeys (invisible on the screen) and the Install / Uninstall buttons at the bottom (wrong style; became `activate` on / off as the first row of `[2]`), a description row under every preference row (moved into `?` help), `.screenrc setenv LOCKPRG` (measured: does not work), a global style setting (colours became each saver's own),
Enter in the sidebar to activate (chosen at preference › profile instead, with the `a` key added later), 12-hour AM/PM, a colon in the time, glyphs with diagonals,
tmux's default `bind L` (clashes with the user's own keys; a command alias instead, and `bind-key` for those who want one), a session-level tmux lock (switch session and you are past it; the whole server instead),
promoting the idle lock to the whole server (a second screen would be locked by the other), one PIN unlocking every client (needs polling; each types its own),
pure write-as-you-set integration (a typo in conf makes a file), pure buttons with no sync (a changed value leaves the file stale), a status row inside `[2]` (the status is the `activate` row), a plain ` · `-separated title (the capsule chain instead), a PIN check on preview too, installed / uninstalled in the title capsule and the kind chained after the title (the status is the `activate` row; the kind moved to a capsule of its own top right and then went too, the classification being noise), the config path on the right of `[2]`'s bottom edge (there from the first version, not a family convention), the lock looking its session up from the tty (list-clients is empty while locked, display-message -c returns the wrong session),
a VT terminal emulator route for custom (one dependency more, fidelity and performance both to prove; passthrough instead), auto-restarting the custom program (its lifetime is not interfered with), a black screen with red text (the board instead), calling the ending COMPLETED / ERROR and then DONE / ERROR (the exit code written as it is), freezing the picture under the PIN box on locku's background (no animation under the box), shrinking the terminal a column and back when the box closes to force a repaint (with the picture not frozen there is nothing to repaint), SIGWINCH to a program that is drawing (cmatrix flashes and starts over), keeping a command list parallel to the block for tmux's live apply (the whole block by `source-file` instead), a forgot-my-PIN entry on the lock screen, recovery codes, an emptied `pin_hash` as the reset, a global `P` on `[1]` (`p` is the row under the cursor), native Windows.

## Layout

```
locku/
├── cmd/locku/          entry point: lock / pin reset / version / the settings TUI; argv[0] SCREEN-LOCK
├── internal/
│   ├── config/         config.yaml in and out: fail open, atomic writes, 0600, the bcrypt PIN, pin_hash re-read, NewPIN, the data directory
│   ├── custom/         the custom saver: the program on a pty, its output through the screen writer, the PIN box over the picture, the keys kept for locku, the word for its ending (2026-09-25)
│   ├── login/          the login-password check for `locku pin reset`: su on a pty (2026-09-25)
│   ├── saver/          the content: clock's two times × five dates × row / column, the tick; dino's runners, scenes, obstacles and automatic jumps; custom's ending Word
│   ├── setup/          the managed block: tmux.conf (a running server gets it whole by source-file), screenrc, the shell rc
│   ├── tmux/           raising / clearing the @locked flag on tmux while locked
│   ├── ui/             the renderer (font / canvas / reveal), the lock screen, the PIN prompt, the settings TUI and its floats
│   └── version/        the version string
├── e2e/                pty end-to-end: tmux_attach.py, custom_lock.py, screen_lock.py
└── docs/               function.md, ui.md, ux.md, icon.svg
```

## Development

```bash
make check                                       # fmt-check + vet + go test -race
LOCKU_DUMP=1 go test ./internal/ui -run TestDump -v   # prints the screen at every size
make lock                                        # build and lock this terminal
make e2e                                         # end to end: a real tmux (e2e/tmux_attach.py, its own TMUX_TMPDIR), the custom saver (e2e/custom_lock.py), a real screen (e2e/screen_lock.py, its own SCREENDIR); your servers and sessions are not touched
```

All TUI behaviour is verified by programmatic model tests (no tty needed); `make check` runs with the race detector (custom's pump and screen writer, login's su and the custom lock's prompt each have a goroutine). The tmux / custom / screen acceptance (`docs/function.md` §12)
is done by python pty harnesses running the real binary: prefix+d, prefix+c and Ctrl+C swallowed while locked, the client back after the right PIN, an attach while locked locked too, a running server swapped whole when the screen switches to lock-session, custom's box over the animation and the program killed clean, screen's `C-a x` and the `bind` key reaching locku, `idle` locking by itself, a running session getting the settings at once, the process ending when the tty closes.

Releases are the family's: push a `v*` tag, GitHub Actions runs the tests on both platforms, goreleaser builds the archives and updates the Homebrew tap, and the release notes are the matching section of `CHANGELOG.md`.

## Next steps

1. An eight-hour CPU / memory watch (the last item of §12).
