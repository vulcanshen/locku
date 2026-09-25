#!/usr/bin/env python3
"""End to end, on a pty with the real binary (function.md §5.6): a custom
saver's program has the screen and is killed when the lock ends; with a
PIN, a key holds its picture back and puts the prompt up, Esc gives it
the screen again, the PIN ends everything; a program that ends — with a
code, with 0, or none set — is the word on locku's board, and the lock
stays; cmatrix, when it is there, runs and is killed like the rest.

    make e2e                # builds the binary and runs this
    e2e/custom_lock.py ./locku

Needs python3; touches nothing of yours: its own HOME and config.
"""
import os, pty, sys, time, fcntl, termios, struct, subprocess, threading, signal, tempfile, shutil

LOCKU = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else "./locku")
work = tempfile.mkdtemp(prefix="locku-e2e-custom-")
cfgDir = os.path.join(work, "cfg")
os.makedirs(cfgDir)
cfgFile = os.path.join(cfgDir, "config.yaml")
env = {"HOME": work, "PATH": os.environ["PATH"], "TERM": "xterm-256color", "SHELL": "/bin/sh",
       "LOCKU_CONFIG": cfgDir, "USER": os.environ.get("USER", "u")}
MARK = "LOCKU-E2E-MARK"
PIXEL = "".encode()   # the board's cell
PROMPT = "".encode()  # the PIN prompt's lock


def config(command, pin_hash=None):
    """A file whose active profile is a custom saver running command."""
    with open(cfgFile, "w") as f:
        f.write('profile: run\nprofiles:\n  - name: run\n    saver: custom\n    command: ' + repr(command) + '\n')
        if pin_hash:
            f.write('pin_hash: ' + repr(pin_hash) + '\n')


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


def output(out):
    return b"".join(out)


def alive(pid):
    try:
        os.kill(pid, 0)
    except OSError:
        return False
    return os.waitpid(pid, os.WNOHANG)[0] == 0


def wait_exit(pid, secs):
    for _ in range(int(secs * 10)):
        if not alive(pid):
            return True
        time.sleep(0.1)
    return False


def kill(pid):
    try:
        os.kill(pid, signal.SIGKILL)
    except OSError:
        pass


def running(pattern):
    return subprocess.run(["pgrep", "-f", pattern], capture_output=True).returncode == 0


ok = True


def check(label, cond):
    global ok
    ok = ok and bool(cond)
    print(("ok   " if cond else "FAIL ") + label)


def lock():
    pid, fd, out = spawn([LOCKU, "lock"])
    time.sleep(1.5)
    return pid, fd, out


loop = "while :; do printf '%s\\n'; sleep 0.2; done" % MARK

# 1. No PIN: the program has the screen; any key ends the lock, and the
# program with it.
config(loop)
pid, fd, out = lock()
check("the program's output is on the terminal", MARK.encode() in output(out))
check("the program is running", running(MARK))
os.write(fd, b"x")
check("any key ends the lock without a PIN", wait_exit(pid, 3))
time.sleep(0.5)
check("the program is killed with it", not running(MARK))

# 2. A PIN, set on the settings screen: preference's first row, the new
# PIN twice.
pid, fd, out = spawn([LOCKU])
time.sleep(1.2)
for k in (b"G", b"2", b"\r", b"1234", b"\r", b"1234", b"\r", b"\x1b", b"q"):
    os.write(fd, k)
    time.sleep(0.5)
wait_exit(pid, 3)
kill(pid)
text = open(cfgFile).read()
check("the settings screen set a PIN", "pin_hash:" in text)
pin_hash = [l.split(":", 1)[1].strip().strip("'\"") for l in text.splitlines() if l.startswith("pin_hash:")][0]

# 3. With the PIN: a key holds the picture back and puts the prompt up;
# Esc gives the program the screen again; the PIN ends everything.
config(loop, pin_hash)
pid, fd, out = lock()
os.write(fd, b"x")
time.sleep(1.0)
o = output(out)
check("a key puts the PIN prompt up", PROMPT in o)
check("over the picture: no ground of locku's, no switch of screens", PIXEL not in o and o.count(b"\x1b[?1049l") == 0)
c1 = o.count(MARK.encode())
time.sleep(0.8)
c2 = output(out).count(MARK.encode())
check("the picture is held back under the prompt", c2 == c1 and c1 > 0)
os.write(fd, b"\x1b")
time.sleep(1.2)
c3 = output(out).count(MARK.encode())
check("Esc gives the program the screen again", c3 > c2)
check("the lock is still up", alive(pid) and running(MARK))
os.write(fd, b"x")
time.sleep(0.8)
os.write(fd, b"1234\r")
check("the PIN ends the lock", wait_exit(pid, 3))
time.sleep(0.5)
check("and the program with it", not running(MARK))

# 4. A program that ends is EXIT and its code on the board, with why,
# and the lock stays — 0 included; none set is NONE. The PIN ends each.
for command, note in (("echo boom >&2; exit 3", b"custom saver: exit 3"), ("exit 0", b"custom saver exited 0"), ("", b"custom saver: no command")):
    config(command, pin_hash)
    pid, fd, out = spawn([LOCKU, "lock"])
    time.sleep(2.5)
    o = output(out)
    check("%r: the word on locku's board" % command, PIXEL in o)
    check("%r: with why, on the status row" % command, note in o and (command != "echo boom >&2; exit 3" or b"boom" in o))
    check("%r: the lock stays" % command, alive(pid))
    os.write(fd, b"x")
    time.sleep(0.8)
    os.write(fd, b"1234\r")
    check("%r: the PIN ends it" % command, wait_exit(pid, 3))
    kill(pid)

# 5. cmatrix, when there is one: it runs, and is killed like the rest.
if shutil.which("cmatrix"):
    config("exec cmatrix -b -u 5", pin_hash)
    pid, fd, out = lock()
    time.sleep(1.0)
    check("cmatrix draws", len(output(out)) > 2000 and running("cmatrix -b -u 5"))
    os.write(fd, b"x")
    time.sleep(0.8)
    o = output(out)
    check("a key puts the prompt over cmatrix", PROMPT in o)
    os.write(fd, b"1234\r")
    check("the PIN ends the lock", wait_exit(pid, 3))
    time.sleep(0.5)
    check("and cmatrix with it", not running("cmatrix -b -u 5"))
else:
    print("skip cmatrix is not on PATH")

print("ALL OK" if ok else "SOMETHING FAILED")
sys.exit(0 if ok else 1)
