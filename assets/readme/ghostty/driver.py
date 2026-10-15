"""Runs one program in a pty inside a real Ghostty window and types a scripted scene into it.

Ghostty starts this script as its command (record.py writes the config). The program runs in a pty of its own, its
output goes to the Ghostty window untouched, and the keys come from the scene: the real keyboard is dropped (only Ghostty's
answers to the program's queries go through), so the recording is the same every time and nothing typed by hand reaches the program.

    driver.py <env.json> <window.json> <sync-dir>

env.json is the demo environment (demo-env.sh). window.json has "name", "argv", optional "cwd", "setup" (steps run
before the recording) and "steps". The files in <sync-dir> coordinate the windows and the recorder:
<name>.ready once setup is done, "go" when the recording starts, "<signal>" for {"post": ...} and {"await": ...}.
"""

from __future__ import annotations

import fcntl
import json
import os
import pty
import re
import select
import signal
import sys
import termios
import time
import tty
from collections.abc import Callable

ANSI = re.compile(rb"\x1b\[[0-9;?<>=]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[P^_][^\x1b]*\x1b\\|\x1b[@-_]")
KEYS = {
    "enter": b"\r",
    "esc": b"\x1b",
    "tab": b"\t",
    "space": b" ",
    "backspace": b"\x7f",
    "up": b"\x1b[A",
    "down": b"\x1b[B",
    "right": b"\x1b[C",
    "left": b"\x1b[D",
    "ctrl+a": b"\x01",
    "ctrl+b": b"\x02",
    "ctrl+c": b"\x03",
    "ctrl+g": b"\x07",
    "ctrl+q": b"\x11",
    "ctrl+s": b"\x13",
    "ctrl+t": b"\x14",
    "ctrl+u": b"\x15",
}
# What Ghostty answers to the program's queries (colors, cursor position, device attributes, kitty graphics, modes,
# capabilities): forwarded, so the program sees the real terminal. Everything else from the keyboard is dropped.
REPLY = re.compile(
    rb"\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)"  # OSC (colors)
    rb"|\x1b_G[^\x1b]*\x1b\\"  # kitty graphics
    rb"|\x1bP[^\x1b]*\x1b\\"  # DCS (XTVERSION, XTGETTCAP)
    rb"|\x1b\[\?[0-9;]*[cu]"  # device attributes, kitty keyboard flags
    rb"|\x1b\[\??[0-9;]*\$y"  # mode reports
    rb"|\x1b\[[0-9]+;[0-9]+R"  # cursor position
    rb"|\x1b\[[0-9;]*t"  # window and cell size
)
WAIT_TIMEOUT = 20.0
KEY_GAP = 0.35
TYPE_SPEED = 0.06


class Window:
    """The program in its pty, the screen text seen so far and the sync folder."""

    def __init__(self, fd: int, sync: str) -> None:
        self.fd = fd
        self.sync = sync
        self.out = sys.stdout.fileno()
        self.keyboard = sys.stdin.fileno()
        self.text = bytearray()  # recent output without escape sequences, for "wait"
        self.alive = True

    def pump(self, seconds: float) -> None:
        """Copies the program's output to Ghostty for a while and drops whatever the real keyboard sends."""
        end = time.monotonic() + seconds
        while self.alive:
            left = end - time.monotonic()
            if left <= 0:
                return
            ready, _, _ = select.select([self.fd, self.keyboard], [], [], min(left, 0.02))
            if self.keyboard in ready:
                for reply in REPLY.findall(os.read(self.keyboard, 4096)):
                    os.write(self.fd, reply)
            if self.fd in ready:
                try:
                    data = os.read(self.fd, 65536)
                except OSError:
                    data = b""
                if not data:
                    self.alive = False
                    return
                os.write(self.out, data)
                self.text.extend(ANSI.sub(b"", data))
                del self.text[:-20000]

    def until(self, check: Callable[[], bool], what: str, timeout: float) -> None:
        end = time.monotonic() + timeout
        while self.alive and not check():
            if time.monotonic() > end:
                raise SystemExit(f"timeout waiting for {what}")
            self.pump(0.05)

    def send(self, data: bytes, pause: float) -> None:
        os.write(self.fd, data)
        self.pump(pause)

    def step(self, step: dict) -> None:
        if "await" in step:
            path = os.path.join(self.sync, step["await"])
            self.until(lambda: os.path.exists(path), step["await"], step.get("timeout", 120))
        if "wait" in step:
            rx = re.compile(step["wait"].encode())
            self.until(lambda: bool(rx.search(bytes(self.text))), step["wait"], step.get("timeout", WAIT_TIMEOUT))
        if step.get("clear"):
            self.text.clear()
        if "sleep" in step:
            self.pump(step["sleep"])
        for key in step.get("keys", []):
            self.send(KEYS.get(key, key.encode()), step.get("gap", KEY_GAP))
        for ch in step.get("type", ""):
            self.send(ch.encode(), step.get("speed", TYPE_SPEED))
        if "raw" in step:
            self.send(step["raw"].encode(), 0.05)
        if "post" in step:
            open(os.path.join(self.sync, step["post"]), "w").close()


def resize(fd: int, out: int) -> None:
    fcntl.ioctl(fd, termios.TIOCSWINSZ, fcntl.ioctl(out, termios.TIOCGWINSZ, b"\0" * 8))


def main() -> int:
    env_file, window_file, sync = sys.argv[1:4]
    # what Ghostty says about itself (TERM, TERMINFO, TERM_PROGRAM, GHOSTTY_*), then the demo environment
    env = {k: v for k, v in os.environ.items() if k.startswith(("TERM", "GHOSTTY"))}
    with open(env_file) as f:
        env.update(json.load(f))
    with open(window_file) as f:
        spec = json.load(f)
    tty.setraw(sys.stdin.fileno())
    # Ghostty titles a window with the folder of its command: the demo folder, never the real home
    os.chdir(spec.get("cwd", env["HOME"]))
    pid, fd = pty.fork()
    if pid == 0:
        os.execvpe(spec["argv"][0], spec["argv"], env)
    out = sys.stdout.fileno()
    resize(fd, out)
    signal.signal(signal.SIGWINCH, lambda *_: resize(fd, out))

    win = Window(fd, sync)
    for step in spec.get("setup", []):
        win.step(step)
    open(os.path.join(sync, spec["name"] + ".ready"), "w").close()
    go = os.path.join(sync, "go")
    win.until(lambda: os.path.exists(go), "go", 120)
    for step in spec["steps"]:
        if not win.alive:
            break
        win.step(step)
    while win.alive:
        win.pump(0.1)
    os.waitpid(pid, 0)
    open(os.path.join(sync, spec["name"] + ".done"), "w").close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
