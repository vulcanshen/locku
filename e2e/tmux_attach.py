#!/usr/bin/env python3
"""End to end, on a real tmux with the real binary (function.md §12):
the settings screen's Integration › tmux › activate writes the block,
the bind-key typed there included; on a server started on it, `locku`
locks every client and marks the server; a client attaching to ANY
session meanwhile is locked by the hook; unlocking clears the mark, and
a client attaching then is not locked; a terminal that dies under the
lock — prefix l having locked it — leaves the mark, and the next client
in meets the lock; with lock-session chosen on the screen, the mark is
one session's and another session is left alone; activate off takes it
all out. No PIN is
set, so any key unlocks — the PIN itself is the unit tests' business.

    make e2e            # builds the binary and runs this
    e2e/tmux_attach.py ./locku

Needs tmux and python3; touches nothing of yours: its own socket, its own
TMUX_TMPDIR, its own HOME and config.
"""
import os, pty, sys, time, fcntl, termios, struct, subprocess, threading, signal, tempfile

LOCKU = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else "./locku")
work = tempfile.mkdtemp(prefix="locku-e2e-")
binDir = os.path.join(work, "bin")
os.makedirs(binDir)
os.symlink(LOCKU, os.path.join(binDir, "locku"))
cfgDir = os.path.join(work, "cfg")
os.makedirs(cfgDir)
conf = os.path.join(work, "tmux.conf")
with open(os.path.join(cfgDir, "config.yaml"), "w") as f:
    f.write('tmux:\n  conf: "' + conf + '"\n')
# TMUX_TMPDIR keeps every tmux here away from the user's own server.
env = {"HOME": work, "PATH": binDir + ":" + os.environ["PATH"], "TERM": "xterm-256color", "SHELL": "/bin/sh",
       "LOCKU_CONFIG": cfgDir, "TMUX_TMPDIR": work, "USER": os.environ.get("USER", "u")}
SOCK = "locku-e2e"


def spawn(argv):
    """argv on a pty of its own, 80×24, its output collected; the terminal
    queries Bubble Tea sends at start-up are answered, as a terminal would."""
    pid, fd = pty.fork()
    if pid == 0:
        os.environ.clear()
        os.environ.update(env)
        os.execvp(argv[0], argv)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 24, 80, 0, 0))
    out = []

    def drain():
        while True:
            try:
                b = os.read(fd, 4096)
            except OSError:
                return
            if not b:
                return
            out.append(b)
            if b"\x1b]11;?" in b:
                os.write(fd, b"\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\")
            if b"\x1b[6n" in b:
                os.write(fd, b"\x1b[1;1R")
    threading.Thread(target=drain, daemon=True).start()
    return pid, fd, out


def tmux(*a):
    r = subprocess.run(["tmux", "-L", SOCK] + list(a), env=env, capture_output=True, text=True)
    return (r.stdout + r.stderr).strip()


def board(out):
    """The LED board is on the pty: the square glyph, and the status row."""
    o = b"".join(out)
    return b"\xef\x83\x88" in o and b"locked since" in o


def marked():
    return tmux("show", "-gv", "@locked") == "1"


def locks_running():
    return subprocess.run(["pgrep", "-f", "locku lock -S"], capture_output=True).returncode == 0


def unlock(fd):
    os.write(fd, b"x")  # no PIN: any key unlocks
    time.sleep(1.2)


def kill(pid):
    for sig in (lambda p: os.killpg(os.getpgid(p), signal.SIGKILL), lambda p: os.kill(p, signal.SIGKILL)):
        try:
            sig(pid)
        except Exception:
            pass


ok = True


def check(label, cond):
    global ok
    ok = ok and bool(cond)
    print(("ok   " if cond else "FAIL ") + label)


def settings(*keys):
    """The settings screen on a pty, the keys pressed one by one, then q.
    Integration › tmux is two rows above preference: G, k, k."""
    pid, fd, out = spawn([LOCKU])
    time.sleep(1.2)
    for k in keys:
        os.write(fd, k)
        time.sleep(0.5)
    os.write(fd, b"q")
    time.sleep(0.8)
    kill(pid)
    return b"".join(out)


# 0. Activate from the screen: G to preference, k k up to tmux, into
# [2], down past the path, the lock and the idle time to bind-key and l
# typed into it, back up to activate, Enter, and Enter on the confirm.
screen = settings(b"G", b"k", b"k", b"2", b"j", b"j", b"j", b"j", b"\r", b"l", b"\r", b"g", b"g", b"\r", b"\r")
text = open(conf).read() if os.path.exists(conf) else ""
check("activate from the settings screen wrote the block, every line marked, the lock command absolute, the key bound",
      "# >>> locku >>>" in text and "client-attached[90]" in text and 'lock-command "/' in text
      and "socket_path" in text and "locku=lock-server" in text and "bind-key l lock-server" in text
      and all("# locku" in l for l in text.splitlines() if l and not l.startswith("#")))
check("the screen said so", b"wrote" in screen)

subprocess.run(["tmux", "-L", SOCK, "kill-server"], env=env, stderr=subprocess.DEVNULL)
pa, fa, oa = spawn(["tmux", "-L", SOCK, "-f", conf, "new-session", "-s", "t"])
time.sleep(1.5)
tmux("new-session", "-d", "-s", "u")  # a second session, nobody on it
check("the server took the block: alias, hooks and the key",
      "locku=lock-server" in tmux("show", "-s", "command-alias") and "client-attached[90]" in tmux("show-hooks", "-g")
      and "lock-server" in tmux("list-keys", "-T", "prefix"))

# 1. `locku` locks A and marks the server.
tmux("locku")
time.sleep(1.8)
check("A shows the board after locku", board(oa))
check("the server is marked @locked", marked())

# 2. B attaches to the same session meanwhile, E to the other: both locked.
pb, fb, ob = spawn(["tmux", "-L", SOCK, "attach", "-t", "t"])
pe, fe, oe = spawn(["tmux", "-L", SOCK, "attach", "-t", "u"])
time.sleep(2.0)
check("B, attaching to the locked session, shows the board", board(ob))
check("E, attaching to the OTHER session, shows the board too", board(oe))

# 3. A unlocks: the mark goes; B and E stay on their own locks until their own key.
unlock(fa)
check("A's unlock clears the mark", not marked())
check("B and E still show the board until their own key", locks_running())
unlock(fb)
unlock(fe)
time.sleep(0.5)
check("B and E are back too", not locks_running())

# 4. C attaches to the unlocked server: no lock.
pc, fc, oc = spawn(["tmux", "-L", SOCK, "attach", "-t", "u"])
time.sleep(1.5)
check("C, attaching once unlocked, sees no board", not board(oc))
for p in (pc, pb, pe):
    kill(p)
time.sleep(0.5)

# 5. prefix l locks A; A's terminal dies under the lock: the mark stays,
# the next client is locked.
os.write(fa, b"\x02l")
time.sleep(1.5)
check("prefix l locks A and marks the server", marked() and locks_running())
kill(pa)
time.sleep(1.5)
check("the mark survives a terminal that died under the lock", marked())
pd, fd_, od = spawn(["tmux", "-L", SOCK, "attach", "-t", "t"])
time.sleep(2.0)
check("D, attaching after that, shows the board", board(od))
unlock(fd_)
check("D's unlock clears the mark", not marked())
tmux("kill-server")
kill(pd)

# 6. lock-session, chosen on the screen while activate is on: the file is
# rewritten at once — the alias, the key and a session-created hook.
screen = settings(b"G", b"k", b"k", b"2", b"j", b"j", b"\r", b"j", b"\r")
text = open(conf).read()
check("lock-session chosen on the screen: the alias, the key and the session hook are in the file",
      "locku=lock-session" in text and "bind-key l lock-session" in text and "session-created[90]" in text
      and "-t '#{session_id}'" in text and "lock-server" not in text)
pa, fa, oa = spawn(["tmux", "-L", SOCK, "-f", conf, "new-session", "-s", "t"])
time.sleep(1.5)
tmux("new-session", "-d", "-s", "u")
# A session is named with its colon: a bare name is tried as a window
# name's prefix first, and u's window is called after its shell (measured
# 2026-09-25: -t t found u's window "tmux").
check("each session's lock-command carries its own id",
      "-t '$0'" in tmux("show", "-t", "t:", "-v", "lock-command") and "-t '$1'" in tmux("show", "-t", "u:", "-v", "lock-command"))
tmux("locku", "-t", "t")
time.sleep(1.8)
check("A shows the board after locku on its session", board(oa))
check("the mark is the session's, not the server's",
      tmux("show", "-t", "t:", "-qv", "@locked") == "1" and tmux("show", "-gqv", "@locked") == "")
pb, fb, ob = spawn(["tmux", "-L", SOCK, "attach", "-t", "t"])
pe, fe, oe = spawn(["tmux", "-L", SOCK, "attach", "-t", "u"])
time.sleep(2.0)
check("B, attaching to the locked session, shows the board", board(ob))
check("E, attaching to the OTHER session, is left alone", not board(oe))
unlock(fa)
check("A's unlock clears the session's mark", tmux("show", "-t", "t:", "-qv", "@locked") == "")
unlock(fb)
time.sleep(0.5)
check("B is back too", not locks_running())
tmux("kill-server")
for p in (pa, pb, pe):
    kill(p)

# 7. Deactivate from the screen: activate is the first row, Enter, then
# Enter on the confirm.
screen = settings(b"G", b"k", b"k", b"2", b"\r", b"\r")
check("deactivate from the settings screen took the block out", "locku" not in open(conf).read())
check("the screen said so", b"removed" in screen)

print("ALL OK" if ok else "SOMETHING FAILED")
sys.exit(0 if ok else 1)
