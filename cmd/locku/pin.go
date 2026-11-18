package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
)

// pinReset is `locku pin reset` (user, 2026-09-25):
// the way back from a forgotten PIN, from any shell of the user's own.
// It asks twice — y/N, then the user's login password, the account
// being the one boundary locku has — then makes a new PIN, writes its
// hash over the old one and shows the PIN once, the way elasticsearch
// resets a password: never an empty PIN, never a lock left open. A lock
// already up reads the new hash at its next key (ui.LockModel), so the
// new PIN opens it. The reset is logged under ~/.locku/data, without
// the PIN. verify is login.Verify, or a stand-in under test.
func pinReset(in *os.File, out, errs io.Writer, verify func(user, password string) bool) int {
	if !term.IsTerminal(in.Fd()) {
		fmt.Fprintln(out, "locku pin reset: needs a terminal to ask on")
		return 2
	}
	// A file of locku's that cannot be read — config.yaml, its pin_hash no
	// hash included, or profiles.yaml — would be read as the defaults and
	// written back over it. With no screen there is no MOVE ASIDE to ask,
	// so the reset does as its Esc does: nothing touched, nothing printed,
	// the path and the reason on stderr (tdp E3). A file that is not there
	// is made.
	for _, f := range []struct {
		path string
		err  error
	}{{config.Path(), config.Unreadable()}, {config.ProfilesPath(), config.UnreadableProfiles()}} {
		if f.err != nil {
			fmt.Fprintf(errs, "locku pin reset: %s can't be read: %v\n", foldHome(f.path), f.err)
			return 1
		}
	}
	cfg, note := config.Load()
	// A value or a profile that did not fit is no file that cannot be read:
	// with no screen too it is kept aside and written again, the sentence on
	// stderr (tdp E3).
	for _, said := range keepAside(cfg) {
		fmt.Fprintln(errs, "locku pin reset: "+said)
	}
	// A profile config.yaml names and profiles.yaml has not stays as it is,
	// said on stderr (tdp E3, a value naming an entry that is not there).
	if note != "" {
		fmt.Fprintln(errs, "locku pin reset: "+note)
	}
	for _, said := range cfg.UnusableSaid() {
		fmt.Fprintln(errs, "locku pin reset: "+said)
	}
	fmt.Fprint(out, "Reset the PIN? A new one is made and shown here once; the old one stops working at once. [y/N] ")
	line, _ := bufio.NewReader(in).ReadString('\n')
	if !strings.EqualFold(strings.TrimSpace(line), "y") {
		fmt.Fprintln(out, "left as it is")
		return 1
	}
	who := username()
	fmt.Fprintf(out, "Password for %s: ", who)
	pw, err := term.ReadPassword(in.Fd())
	fmt.Fprintln(out)
	if err != nil || !verify(who, string(pw)) {
		logReset(who, "refused: the password is not the account's")
		fmt.Fprintln(out, "locku pin reset: that is not your login password; nothing changed")
		return 1
	}
	pin, err := config.NewPIN()
	if err != nil {
		fmt.Fprintln(out, "locku pin reset:", err)
		return 1
	}
	if err := cfg.SetPIN(pin); err != nil {
		fmt.Fprintln(out, "locku pin reset:", err)
		return 1
	}
	if err := config.Save(cfg); err != nil {
		fmt.Fprintln(out, "locku pin reset:", err)
		return 1
	}
	logged := logReset(who, "a new PIN written")
	fmt.Fprintf(out, "New PIN: %s\n  written to %s — a lock already up takes it at its next key; change it on the settings screen when you like\n  logged in %s\n", pin, foldHome(config.Path()), foldHome(logged))
	return 0
}

// keepAside is what a start without the settings screen does with a file
// that had a value or a profile that did not fit (tdp E3), as the settings
// screen does: the file copied whole to the next free .bak and written
// again with the defaults. It returns what to say, a sentence a file —
// the lib's — or, when the copy cannot be made, why, the file left as it
// was and the defaults for this run only.
func keepAside(cfg config.Config) []string {
	var said []string
	for _, path := range cfg.MisfitFiles() {
		var ms []terminu.Misfit
		for _, m := range cfg.MisfitsIn(path) {
			ms = append(ms, terminu.Misfit{Key: m.Key, Was: m.Was, Now: m.Now})
		}
		copy, err := terminu.KeepAside(path)
		if err == nil {
			err = config.Rewrite(cfg, path)
		}
		if err != nil {
			said = append(said, err.Error())
			continue
		}
		said = append(said, terminu.DidNotFitText(path, copy, ms...))
	}
	return said
}

// username is who is asking: the account, for su and for the log.
func username() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return os.Getenv("USER")
}

// logReset appends one line to the reset log — when, who, what came of
// it — and returns the log's path. The PIN is never in it.
func logReset(who, what string) string {
	dir := config.DataDir()
	path := filepath.Join(dir, "pin-resets.log")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return path
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return path
	}
	defer f.Close()
	host, _ := os.Hostname()
	fmt.Fprintf(f, "%s pin reset by %s@%s: %s\n", time.Now().Format(time.RFC3339), who, host, what)
	return path
}

// foldHome writes the home directory as ~.
func foldHome(p string) string {
	if h, err := os.UserHomeDir(); err == nil && h != "" && strings.HasPrefix(p, h) {
		return "~" + strings.TrimPrefix(p, h)
	}
	return p
}
