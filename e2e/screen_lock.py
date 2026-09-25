#!/usr/bin/env python3
"""End to end, on a real GNU screen with the real binary (function.md
§12): the settings screen's Integration › screen › activate writes the
block into the screenrc, the bind typed there included, and LOCKPRG into
the shell rc; a screen already running takes the idle time and the key
at once (`screen -X`); on a screen started on the file with LOCKPRG in
its environment, C-a x shows locku's board, and so does the bound key;
idle set to 2 on the screen while activate is on is in the file at once
and locks the running screen by itself; activate off takes it all out —
the files, and the running screen's idle and key. No PIN is set, so any
key unlocks — the PIN itself is the unit tests' business.

    make e2e            # builds the binary and runs this
    e2e/screen_lock.py ./locku

Needs screen and python3; touches nothing of yours: its own SCREENDIR,
its own HOME and config, SHELL=/bin/sh so the shell rc is ~/.profile.
"""
import os, pty, sys, time, fcntl, termios, struct, subprocess, threading, signal, tempfile

LOCKU = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else "./locku")
work = tempfile.mkdtemp(prefix="locku-e2e-screen-")
binDir = os.path.join(work, "bin")
os.makedirs(binDir)
os.symlink(LOCKU, os.path.join(binDir, "locku"))
cfgDir = os.path.join(work, "cfg")
os.makedirs(cfgDir)
rc = os.path.join(work, "screenrc")
with open(rc, "w") as f:
    f.write("startup_message off\n")
with open(os.path.join(cfgDir, "config.yaml"), "w") as f:
    f.write('screen:\n  conf: "' + rc + '"\n')
profile = os.path.join(work, ".profile")
# SCREENDIR keeps every screen here away from the user's own sessions.
sdir = os.path.join(work, "screens")
os.makedirs(sdir, mode=0o700)
env = {"HOME": work, "PATH": binDir + ":" + os.environ["PATH"], "TERM": "xterm-256color", "SHELL": "/bin/sh",
       "LOCKU_CONFIG": cfgDir, "SCREENDIR": sdir, "USER": os.environ.get("USER", "u")}
# What activate writes into ~/.profile: LOCKPRG is locku as PATH finds
# it. A screen started here gets it the way a new shell would.
screenEnv = dict(env, LOCKPRG=os.path.join(binDir, "locku"))


def spawn(argv, e=env):
    """argv on a pty of its own, 80×24, its output collected; the terminal
    queries Bubble Tea sends at start-up are answered, as a terminal would."""
    pid, fd = pty.fork()
    if pid == 0:
        os.environ.clear()
        os.environ.update(e)
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


def screen(*a):
    r = subprocess.run(["screen"] + list(a), env=env, capture_output=True, text=True)
    return (r.stdout + r.stderr).strip()


def board(out):
    """The LED board is on the pty: the square glyph, and the status row."""
    o = b"".join(out)
    return b"\xef\x83\x88" in o and b"locked since" in o


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
    Integration › screen is one row above preference: G, k."""
    pid, fd, out = spawn([LOCKU])
    time.sleep(1.2)
    for k in keys:
        os.write(fd, k)
        time.sleep(0.5)
    os.write(fd, b"q")
    time.sleep(0.8)
    kill(pid)
    return b"".join(out)


# A screen is running before anything is activated — on the file as it
# is, with nothing of locku's in it — so what it takes later, it takes
# live. LOCKPRG is in its environment, as a new shell's would be.
pa, fa, oa = spawn(["screen", "-c", rc, "-S", "t"], screenEnv)
time.sleep(1.5)
check("a screen is running on its own SCREENDIR", ".t\t" in screen("-ls"))

# 0. Activate from the screen: G to preference, k up to screen, into
# [2], down past the path and the idle time to bind and l typed into it,
# back up to activate, Enter, and Enter on the confirm.
shown = settings(b"G", b"k", b"2", b"j", b"j", b"j", b"\r", b"l", b"\r", b"g", b"g", b"\r", b"\r")
text = open(rc).read()
block = text[text.find("# >>> locku >>>"):text.find("# <<< locku <<<")].splitlines()[1:]
check("activate from the settings screen wrote the block, every line marked, the idle time and the key bound, the user's line kept",
      "# >>> locku >>>" in text and "idle 300 lockscreen" in text and "bind l lockscreen" in text
      and text.startswith("startup_message off\n") and block and all("# locku" in l for l in block))
prof = open(profile).read() if os.path.exists(profile) else ""
check("LOCKPRG went into ~/.profile (SHELL=/bin/sh), locku by its absolute path, marked",
      "export LOCKPRG=" + os.path.join(binDir, "locku") in prof and "# locku" in prof)
# The toast is one line of an 80-column screen: the first words show;
# that the running screen took it is checked on the screen itself.
check("the screen said so", b"wrote" in shown)

# 1. C-a x, screen's own key, runs LOCKPRG: locku's board on A.
os.write(fa, b"\x01x")
time.sleep(1.8)
check("A shows the board after C-a x", board(oa))
unlock(fa)
del oa[:]

# 2. The bound key, taken live by the running screen — the file it was
# started on had no bind.
os.write(fa, b"\x01l")
time.sleep(1.8)
check("A shows the board after C-a l, bound live", board(oa))
unlock(fa)
del oa[:]

# 3. A screen started on the file now: the bind is read from the file.
pb, fb, ob = spawn(["screen", "-c", rc, "-S", "u"], screenEnv)
time.sleep(1.5)
os.write(fb, b"\x01l")
time.sleep(1.8)
check("B, started on the file, shows the board after C-a l", board(ob))
unlock(fb)
screen("-S", "u", "-X", "quit")
time.sleep(0.5)
kill(pb)

# 4. idle 2 typed on the screen while activate is on: the file is
# rewritten at once, the running screen takes it, and locks by itself.
shown = settings(b"G", b"k", b"2", b"j", b"j", b"\r", b"\x15", b"2", b"\r")
check("idle 2 chosen on the screen is in the file at once", "idle 2 lockscreen" in open(rc).read())
check("the screen said so", b"wrote" in shown)
time.sleep(2.5)
check("A locks by itself after 2 idle seconds", board(oa))
unlock(fa)
del oa[:]

# 5. Deactivate from the screen: activate is the first row, Enter, then
# Enter on the confirm. The files lose the block; the running screen
# loses the idle timer and the key.
shown = settings(b"G", b"k", b"2", b"\r", b"\r")
check("deactivate from the settings screen took the block out of the screenrc", "locku" not in open(rc).read() and "startup_message off" in open(rc).read())
check("and out of ~/.profile", "LOCKPRG" not in open(profile).read())
check("the screen said so", b"removed" in shown)
# A may have idled into a lock again meanwhile: a key ends it either way.
unlock(fa)
del oa[:]
time.sleep(3.5)
check("A no longer locks by itself", not board(oa))
os.write(fa, b"\x01l")
time.sleep(1.8)
check("C-a l no longer locks A", not board(oa))

screen("-S", "t", "-X", "quit")
time.sleep(0.5)
kill(pa)

print("ALL OK" if ok else "SOMETHING FAILED")
sys.exit(0 if ok else 1)
