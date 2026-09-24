#!/usr/bin/env python3
"""End to end, on a real tmux with the real binary (function.md §12):
`locku setup tmux` writes the block; a server started on it locks its
client with lock-session and marks the session; a client attaching
meanwhile is locked by the hook; unlocking clears the mark, and a client
attaching then is not locked; a terminal that dies under the lock leaves
the mark, and the next client in meets the lock; `setup -d` takes it all
out. No PIN is set, so any key unlocks — the PIN itself is the unit
tests' business.

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
os.symlink(LOCKU, os.path.join(binDir, "locku"))  # the block says `locku lock`
cfgDir = os.path.join(work, "cfg")
os.makedirs(cfgDir)
conf = os.path.join(work, "tmux.conf")
with open(os.path.join(cfgDir, "config.yaml"), "w") as f:
    f.write('tmux_conf: "' + conf + '"\n')
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
    return "@locked 1" in tmux("show", "-t", "t", "@locked")


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


def setup(*a):
    return subprocess.run([LOCKU, "setup"] + list(a), env=env, capture_output=True, text=True).stdout.strip()


print(setup("tmux"))
text = open(conf).read()
check("the block is in the file, every line marked", "# >>> locku >>>" in text and "client-attached[90]" in text
      and "socket_path" in text and all("# locku" in l for l in text.splitlines() if l and not l.startswith("#")))

subprocess.run(["tmux", "-L", SOCK, "kill-server"], env=env, stderr=subprocess.DEVNULL)
pa, fa, oa = spawn(["tmux", "-L", SOCK, "-f", conf, "new-session", "-s", "t"])
time.sleep(1.5)
check("the server took the block: alias and hooks",
      "locku=lock-session" in tmux("show", "-s", "command-alias") and "client-attached[90]" in tmux("show-hooks", "-g"))

# 1. lock-session locks A and marks the session.
tmux("lock-session", "-t", "t")
time.sleep(1.8)
check("A shows the board after lock-session", board(oa))
check("the session is marked @locked", marked())

# 2. B attaches meanwhile and is locked by the hook.
pb, fb, ob = spawn(["tmux", "-L", SOCK, "attach", "-t", "t"])
time.sleep(2.0)
check("B, attaching to the locked session, shows the board", board(ob))

# 3. A unlocks: the mark goes; B is on its own lock until its own key.
unlock(fa)
check("A's unlock clears the mark", not marked())
check("B still shows the board until its own key", locks_running())
unlock(fb)
time.sleep(0.5)
check("B is back too", not locks_running())

# 4. C attaches to the unlocked session: no lock.
pc, fc, oc = spawn(["tmux", "-L", SOCK, "attach", "-t", "t"])
time.sleep(1.5)
check("C, attaching to the unlocked session, sees no board", not board(oc))
kill(pc)
kill(pb)
time.sleep(0.5)

# 5. A's terminal dies under the lock: the mark stays, the next client is locked.
tmux("lock-session", "-t", "t")
time.sleep(1.5)
kill(pa)
time.sleep(1.5)
check("the mark survives a terminal that died under the lock", marked())
pd, fd_, od = spawn(["tmux", "-L", SOCK, "attach", "-t", "t"])
time.sleep(2.0)
check("D, attaching after that, shows the board", board(od))
unlock(fd_)
check("D's unlock clears the mark", not marked())

# 6. setup -d takes it all out of the file.
print(setup("-d", "tmux"))
check("the block is gone from the file", "locku" not in open(conf).read())

tmux("kill-server")
kill(pd)
print("ALL OK" if ok else "SOMETHING FAILED")
sys.exit(0 if ok else 1)
