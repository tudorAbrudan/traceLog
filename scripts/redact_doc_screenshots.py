#!/usr/bin/env python3
"""
Copy dashboard screenshots into docs/public/screenshots/ with sensitive regions
obscured (IPs, domains, hostnames, container names, log lines, server IDs).

Usage:
  python3 scripts/redact_doc_screenshots.py /path/to/raw/screenshots

Raw filenames must match the keys below (full Cursor export names).
"""
from __future__ import annotations

import sys
from pathlib import Path

from PIL import Image, ImageDraw

REPO_ROOT = Path(__file__).resolve().parent.parent
OUT_DIR = REPO_ROOT / "docs" / "public" / "screenshots"
# Slate panel color close to TraceLog dark UI
FILL = (30, 41, 59)

# (output_name, raw_filename_suffix_or_full, list of [x0,y0,x1,y1])
# Use unique substring of the saved export filename to avoid brittle full names.
SPECS: list[tuple[str, str, list[tuple[int, int, int, int]]]] = [
    (
        "overview.png",
        "12.38.04-3a3e02e1",
        [],
    ),
    (
        "servers.png",
        "12.39.53-ff9b8632",
        [(182, 92, 1000, 218)],
    ),
    (
        "uptime.png",
        "12.39.42-1c6f4dc8",
        [(180, 128, 840, 312)],
    ),
    (
        "logs.png",
        "12.37.34-4dd54611",
        [
            (168, 48, 1008, 128),
            (186, 98, 984, 160),
            (160, 198, 1024, 528),
        ],
    ),
    (
        "docker.png",
        "12.39.34-3dbcdbcb",
        [
            (200, 166, 448, 332),
            (188, 318, 1024, 532),
        ],
    ),
    (
        "docker-light.png",
        "12.40.55-c241bdd3",
        [
            (200, 166, 448, 332),
            (188, 318, 1024, 532),
        ],
    ),
    (
        "processes.png",
        "12.37.22-d84a2a79",
        [(418, 138, 1024, 518)],
    ),
    (
        "alerts.png",
        "12.37.02-c08c1af8",
        [(160, 300, 1024, 410)],
    ),
    (
        "log-sources.png",
        "12.36.30-edf38aca",
        [],
    ),
]


def find_raw(src_dir: Path, needle: str) -> Path:
    for p in src_dir.iterdir():
        if p.suffix.lower() == ".png" and needle in p.name:
            return p
    raise FileNotFoundError(f"No PNG matching {needle!r} in {src_dir}")


def redact_image(im: Image.Image, boxes: list[tuple[int, int, int, int]]) -> Image.Image:
    rgb = im.convert("RGB")
    dr = ImageDraw.Draw(rgb)
    for b in boxes:
        dr.rectangle(b, fill=FILL)
    return rgb


def main() -> None:
    if len(sys.argv) < 2:
        print("Usage: python3 scripts/redact_doc_screenshots.py <raw-screenshots-dir>", file=sys.stderr)
        sys.exit(1)
    src_dir = Path(sys.argv[1]).expanduser().resolve()
    if not src_dir.is_dir():
        print(f"Not a directory: {src_dir}", file=sys.stderr)
        sys.exit(1)

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    for out_name, needle, boxes in SPECS:
        raw = find_raw(src_dir, needle)
        img = Image.open(raw)
        done = redact_image(img, boxes)
        dest = OUT_DIR / out_name
        done.save(dest, "PNG", optimize=True)
        print(f"Wrote {dest.relative_to(REPO_ROOT)} ({raw.name})")


if __name__ == "__main__":
    main()
