#!/usr/bin/env python3
"""Run Nucleus describe and optional executable plan preflight."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path


def run_json(command: list[str]) -> dict:
    completed = subprocess.run(
        command,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if completed.returncode != 0:
        sys.stderr.write(completed.stderr)
        raise SystemExit(completed.returncode)
    try:
        return json.loads(completed.stdout)
    except json.JSONDecodeError as exc:
        sys.stderr.write(f"Command did not return JSON: {' '.join(command)}\n{exc}\n")
        raise SystemExit(2) from exc


def ensure_describe_shape(describe: dict) -> None:
    missing = [
        key
        for key in ("edit_surfaces", "generated_freshness", "capability_graph", "verification")
        if key not in describe
    ]
    if missing:
        sys.stderr.write("describe output missing required keys: " + ", ".join(missing) + "\n")
        raise SystemExit(2)


def write_json(path: Path, value: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dir", default=".", help="Nucleus service directory")
    parser.add_argument("--task", default="", help="Optional task for executable planning")
    parser.add_argument("--out", default="", help="Optional directory for JSON evidence output")
    parser.add_argument("--nucleus-bin", default="nucleus", help="Nucleus CLI executable")
    args = parser.parse_args()

    describe = run_json([args.nucleus_bin, "describe", "--dir", args.dir, "--json", "--pretty"])
    ensure_describe_shape(describe)

    plan = None
    if args.task:
        plan = run_json(
            [
                args.nucleus_bin,
                "plan",
                "--dir",
                args.dir,
                "--task",
                args.task,
                "--json",
                "--executable",
            ]
        )

    if args.out:
        out_dir = Path(args.out)
        write_json(out_dir / "describe.json", describe)
        if plan is not None:
            write_json(out_dir / "plan.json", plan)

    edit_surfaces = describe.get("edit_surfaces") or {}
    verification = describe.get("verification") or {}
    summary = {
        "service": describe.get("service", {}),
        "allowed": edit_surfaces.get("allowed", []),
        "readonly": edit_surfaces.get("readonly", []),
        "forbidden": edit_surfaces.get("forbidden", []),
        "verification_commands": verification.get("commands", []),
    }
    if plan is not None:
        summary["plan_task_type"] = plan.get("task_type")
        summary["planned_edits"] = plan.get("edits", plan.get("suggested_edits", []))
        summary["blocked_edits"] = plan.get("blocked_edits", [])

    print(json.dumps(summary, indent=2, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
