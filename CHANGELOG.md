# Changelog

## [Unreleased]
### Added
- `locku lock`: the terminal as one LED board — every cell a Nerd Font square, dark in the saver's `bg`, lit in its `fg` — spelling the clock in a 5 × 7 pixel font at the largest whole scale that fits, the pixels stretched taller by up to half where the rows allow; a change lands as a shuffled reveal of only the pixels that differ.
- A saver's `layout`: `row` (the time on one line, the date on the next) or `column` (`HH` over `MM` over `SS`, the date's parts under them), which makes the digits several times bigger.
- Any key raises the PIN prompt; the key itself is not input. `Enter` unlocks, `Esc` goes back; a wrong PIN holds the prompt red for a second; `lockout_after` wrong PINs in a row start a `lockout_seconds` countdown; `prompt_timeout` seconds of silence close the prompt.
- Without a PIN the lock is a screensaver: any key ends it, and the status row says so. A config file that cannot be read fails open the same way.
- The status row: `user@host · locked since HH:MM`, `show_status: false` to hide it.
- Signals are ignored, the tty is held raw, a panic puts the lock back up; the process ends only for the right PIN, any key with no PIN, or the terminal going away.
- `locku`: the settings screen — a saver's layout, time and date shapes, and its two colours by RGB sliders (a draft until `S` saves it, `R` drops it, `q` asks when one is unsaved); savers duplicated, renamed, deleted, previewed one by one with `p`; preference: the PIN (set, change, clear), the active saver, `show_status`, `prompt_timeout`, the lockout. Everything but the colours is written at once. `P` previews the lock in place.
- `locku setup [tmux|screen]`: a managed block in `~/.tmux.conf` (applied to a running server too) and in `~/.screenrc` plus `LOCKPRG` in the shell rc.
- `SCREEN-LOCK` as argv[0] runs the lock, which is how LOCKPRG is called by screen.
