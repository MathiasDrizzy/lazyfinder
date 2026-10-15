"""Records the README GIFs in real Ghostty windows.

Run from the repository root on macOS, with Ghostty, ffmpeg and uv installed:

    uv run --with pyobjc-framework-Quartz --with pyobjc-framework-Cocoa python assets/readme/ghostty/record.py [scene ...]

Each scene in assets/readme/ghostty/scenes is one GIF. For each scene the script prepares a fresh demo environment
(demo-env.sh: temporary HOME, fake openers, a herdr with the lazyfinder plugin), applies the scene's "config" to the
demo config.json of lazyfinder, opens one Ghostty window per entry of "windows" side by side, and
driver.py runs the window's program and types its steps. The windows are captured straight from the window server
(the content of each window, not the screen), cut to the terminal and encoded with ffmpeg.

Ghostty takes the keyboard focus while it records; the keys typed by hand are dropped by driver.py.
"""

from __future__ import annotations

import json
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from pathlib import Path

import Quartz

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[2]
OUT_DIR = REPO / "assets" / "readme"
FPS = 20
# points cut off each side of the window: its rounded corners. The GIF is the terminal from edge to edge, like the
# VHS ones of lazymark; the window padding is larger by as much, so the text keeps its margin.
INSET = 14
GAP = 14  # points between two windows
ORIGIN = (60, 70)  # where the first window opens on the screen
MAX_SECONDS = 120
GHOSTTY = {
    "font-family": "JetBrainsMono Nerd Font",
    "font-size": "16",
    "theme": "Catppuccin Mocha",
    "window-padding-x": "28",
    "window-padding-y": "24",
    "window-padding-color": "extend",
    "window-padding-balance": "true",
    "window-save-state": "never",
    "confirm-close-surface": "false",
    "quit-after-last-window-closed": "true",
    "macos-titlebar-style": "hidden",
    "cursor-style-blink": "false",
    "shell-integration": "none",
}
ENV_KEYS = (
    "HOME",
    "PATH",
    "LANG",
    "MICRO_TRUECOLOR",
    "COLORTERM",
    "SHELL",
    "USER",
    "LOGNAME",
    "TMPDIR",
    "BASH_SILENCE_DEPRECATION_WARNING",
)


def demo_env() -> dict[str, str]:
    """Sources demo-env.sh in bash (it rebuilds the demo HOME) and returns the environment it leaves."""
    dump = "import json, os; print(json.dumps(dict(os.environ)))"
    out = subprocess.run(
        ["bash", "-c", f'source assets/readme/demo-env.sh >/dev/null 2>&1 || exit 1; python3 -c "{dump}"'],
        cwd=REPO,
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    env = json.loads(out.strip().splitlines()[-1])
    keep = {k: v for k, v in env.items() if k in ENV_KEYS or k.startswith(("XDG_", "GIT_"))}
    keep["PS1"] = "\\[\\e[38;2;137;180;250m\\]\\W\\[\\e[0m\\] $ "
    return keep


def ghostty_windows() -> dict[int, dict]:
    infos = Quartz.CGWindowListCopyWindowInfo(Quartz.kCGWindowListOptionOnScreenOnly, Quartz.kCGNullWindowID)
    return {
        w["kCGWindowNumber"]: w
        for w in infos
        if w.get("kCGWindowOwnerName") == "Ghostty" and w.get("kCGWindowLayer") == 0
    }


def open_window(spec: dict, x: int, y: int, work: Path, env_file: Path, sync: Path) -> dict:
    """Opens a new Ghostty instance with one window running driver.py and returns its window info."""
    spec_file = work / f"{spec['name']}.json"
    spec_file.write_text(json.dumps(spec))
    conf = dict(GHOSTTY)
    conf.update(
        {
            "window-width": str(spec["cols"]),
            "window-height": str(spec["rows"]),
            "window-position-x": str(x),
            "window-position-y": str(y),
            # a fixed title: without one Ghostty shows the folder it was started from, outside the demo
            "title": spec.get("title", "lazyfinder"),
            "command": f"{sys.executable} -I {HERE / 'driver.py'} {env_file} {spec_file} {sync}",
        }
    )
    conf_file = work / f"{spec['name']}.ghostty"
    conf_file.write_text("".join(f"{k} = {v}\n" for k, v in conf.items()))
    before = set(ghostty_windows())
    subprocess.run(
        ["open", "-n", "-a", "Ghostty", "--args", "--config-default-files=false", f"--config-file={conf_file}"],
        check=True,
    )
    end = time.monotonic() + 20
    while time.monotonic() < end:
        new = [w for n, w in ghostty_windows().items() if n not in before and w["kCGWindowBounds"]["Width"] > 150]
        if new:
            return new[0]
        time.sleep(0.1)
    raise SystemExit(f"{spec['name']}: no Ghostty window")


def wait_files(paths: list[Path], timeout: float) -> None:
    end = time.monotonic() + timeout
    while not all(p.exists() for p in paths):
        if time.monotonic() > end:
            raise SystemExit("timeout waiting for " + ", ".join(str(p.name) for p in paths if not p.exists()))
        time.sleep(0.05)


class Canvas:
    """The windows drawn where they are on the screen, at one pixel per point, cut INSET points inside their edges."""

    def __init__(self, windows: list[dict]) -> None:
        bounds = [w["kCGWindowBounds"] for w in windows]
        self.left = min(b["X"] for b in bounds) + INSET
        self.top = min(b["Y"] for b in bounds) + INSET
        right = max(b["X"] + b["Width"] for b in bounds) - INSET
        bottom = max(b["Y"] + b["Height"] for b in bounds) - INSET
        self.width = int(right - self.left + 1) // 2 * 2
        self.height = int(bottom - self.top + 1) // 2 * 2
        self.windows = windows
        space = Quartz.CGColorSpaceCreateWithName(Quartz.kCGColorSpaceSRGB)
        self.ctx = Quartz.CGBitmapContextCreate(
            None, self.width, self.height, 8, self.width * 4, space, Quartz.kCGImageAlphaPremultipliedLast
        )
        Quartz.CGContextSetInterpolationQuality(self.ctx, Quartz.kCGInterpolationHigh)

    def frame(self) -> bytes:
        ctx = self.ctx
        for w in self.windows:
            b = w["kCGWindowBounds"]
            img = Quartz.CGWindowListCreateImage(
                Quartz.CGRectNull,
                Quartz.kCGWindowListOptionIncludingWindow,
                w["kCGWindowNumber"],
                Quartz.kCGWindowImageBoundsIgnoreFraming | Quartz.kCGWindowImageBestResolution,
            )
            if img is None:
                continue
            x = b["X"] - self.left
            y = self.height - (b["Y"] - self.top) - b["Height"]
            Quartz.CGContextDrawImage(ctx, Quartz.CGRectMake(x, y, b["Width"], b["Height"]), img)
        img = Quartz.CGBitmapContextCreateImage(ctx)
        return bytes(Quartz.CGDataProviderCopyData(Quartz.CGImageGetDataProvider(img)))


def record(scene_file: Path) -> None:
    scene = json.loads(scene_file.read_text())
    out = OUT_DIR / scene["output"]
    print(f"── {scene_file.name} → {out.relative_to(REPO)}", flush=True)
    work = Path(tempfile.mkdtemp(prefix="lazyfinder-rec-"))
    sync = work / "sync"
    sync.mkdir()
    env = demo_env()
    env_file = work / "env.json"
    env_file.write_text(json.dumps(env))
    if "config" in scene:  # e.g. the edge of the herdr sidebar for this GIF
        cfg_file = Path(env["XDG_CONFIG_HOME"]) / "lazyfinder" / "config.json"
        cfg = json.loads(cfg_file.read_text())
        cfg.update(scene["config"])
        cfg_file.write_text(json.dumps(cfg, indent=2))
    windows, pids = [], []
    try:
        x, y = ORIGIN
        for spec in scene["windows"]:
            spec = dict(spec, cwd=os.path.join(env["HOME"], spec.get("cwd", "")))
            w = open_window(spec, x, y, work, env_file, sync)
            windows.append(w)
            pids.append(w["kCGWindowOwnerPID"])
            x = int(w["kCGWindowBounds"]["X"] + w["kCGWindowBounds"]["Width"] + GAP)
        wait_files([sync / f"{s['name']}.ready" for s in scene["windows"]], 60)
        time.sleep(0.5)  # the last setup frame reaches the window
        windows = [ghostty_windows()[w["kCGWindowNumber"]] for w in windows]  # final bounds
        encode(Canvas(windows), sync, scene, work / "raw.mkv")
        if out.suffix == ".png":  # a still: the last frame
            last = ["ffmpeg", "-loglevel", "error", "-y", "-sseof", "-0.2", "-i", str(work / "raw.mkv")]
            subprocess.run([*last, "-frames:v", "1", str(out)], check=True)
        else:
            to_gif(work / "raw.mkv", out, scene.get("speed", 1))
    finally:
        for pid in pids:
            try:
                os.kill(pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
        # a herdr scene leaves its server running after Ghostty quits: it is the one of the demo HOME (env has no
        # HERDR_* variable), no other is touched
        subprocess.run(["herdr", "server", "stop"], check=False, env=env, capture_output=True)
        shutil.rmtree(work, ignore_errors=True)
    print(f"   {out.stat().st_size / 1e6:.2f} MB", flush=True)


def activate(windows: list[dict]) -> None:
    """Brings the recorded Ghostty to the front, so its window is drawn focused."""
    from AppKit import NSApplicationActivateIgnoringOtherApps, NSRunningApplication

    for w in windows:
        app = NSRunningApplication.runningApplicationWithProcessIdentifier_(w["kCGWindowOwnerPID"])
        if app is not None:
            app.activateWithOptions_(NSApplicationActivateIgnoringOtherApps)
    time.sleep(0.4)


def encode(canvas: Canvas, sync: Path, scene: dict, raw: Path) -> None:
    """Captures the windows at FPS until the scene posts "end" (or the first window's program exits)."""
    ffmpeg = subprocess.Popen(
        ["ffmpeg", "-loglevel", "error", "-y", "-f", "rawvideo", "-pix_fmt", "rgba"]
        + ["-s", f"{canvas.width}x{canvas.height}", "-r", str(FPS), "-i", "-", "-c:v", "ffv1", str(raw)],
        stdin=subprocess.PIPE,
    )
    stop = [sync / "end", sync / f"{scene['windows'][0]['name']}.done"]
    activate(canvas.windows)
    (sync / "go").touch()
    start = time.monotonic()
    n = 0
    while not any(p.exists() for p in stop) and time.monotonic() - start < MAX_SECONDS:
        ffmpeg.stdin.write(canvas.frame())
        n += 1
        delay = start + n / FPS - time.monotonic()
        if delay > 0:
            time.sleep(delay)
    ffmpeg.stdin.close()
    if ffmpeg.wait() != 0:
        raise SystemExit("ffmpeg failed")


def to_gif(raw: Path, out: Path, speed: float) -> None:
    """speed above 1 plays the recording faster (a tour that shows much in little time)."""
    graph = (
        f"setpts=PTS/{speed},fps={FPS},split[a][b];[a]palettegen=max_colors=256:stats_mode=diff[p];"
        "[b][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle"
    )
    subprocess.run(["ffmpeg", "-loglevel", "error", "-y", "-i", str(raw), "-vf", graph, str(out)], check=True)


def main() -> None:
    names = sys.argv[1:]
    scenes = sorted((HERE / "scenes").glob("*.json"))
    if names:
        scenes = [s for s in scenes if s.stem in names]
    for s in scenes:
        record(s)


if __name__ == "__main__":
    main()
