#!/bin/sh
# locku uninstaller for macOS / Linux.
# Usage: curl -fsSL https://raw.githubusercontent.com/vulcanshen/locku/main/uninstall.sh | sh

set -e

CANDIDATES="$HOME/.local/bin/locku /usr/local/bin/locku"

FOUND=""
for path in $CANDIDATES; do
  if [ -f "$path" ]; then
    FOUND="$path"
    break
  fi
done

if [ -z "$FOUND" ]; then
  echo "locku not found in expected locations."
  echo "Checked: $CANDIDATES"
  exit 1
fi

rm "$FOUND"
echo "removed $FOUND"

# ask offers to remove one directory: the answer is read from the terminal,
# not stdin — under `curl ... | sh` stdin is the script itself, so a plain
# `read` would consume script text (or hit EOF) instead of the keypress. No
# controlling terminal (cron, nohup) -> keep it, the safe default.
ask() {
  dir="$1"; what="$2"
  [ -d "$dir" ] || return 0
  printf "Remove %s in %s? [y/N]: " "$what" "$dir"
  if read -r answer < /dev/tty 2>/dev/null; then
    case "$answer" in
      y|Y|yes|YES) rm -rf "$dir"; echo "removed $dir" ;;
      *) echo "kept $dir" ;;
    esac
  else
    echo ""
    echo "kept $dir (no terminal to confirm on)"
  fi
}

# What the user wrote: the PIN's hash, the savers, the colours.
if [ -n "$LOCKU_CONFIG" ]; then
  CONFIG_DIR="$LOCKU_CONFIG"
elif [ -n "$XDG_CONFIG_HOME" ]; then
  CONFIG_DIR="$XDG_CONFIG_HOME/locku"
else
  CONFIG_DIR="$HOME/.config/locku"
fi
ask "$CONFIG_DIR" "locku settings (the PIN and the savers)"

echo ""
echo "locku uninstalled. The managed blocks 'locku setup' wrote — between"
echo "'# >>> locku >>>' and '# <<< locku <<<' in ~/.tmux.conf, ~/.screenrc and"
echo "your shell rc — are left for you to remove by hand."
