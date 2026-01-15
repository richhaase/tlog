#!/usr/bin/env python3
"""
tlog - Append-Only Task Log for AI Coding Agents

An AI-first task tracker with dependency management.
All mutations are appends - no updates, no deletes.
Git-native, conflict-free by design.
"""

import argparse
import hashlib
import json
import os
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

TLOG_DIR = ".tlog"
EVENTS_DIR = "events"
CONFIG_FILE = "config.json"

# Event types
EVENT_CREATE = "create"
EVENT_STATUS = "status"
EVENT_DEP = "dep"
EVENT_BLOCK = "block"
EVENT_UPDATE = "update"


def get_tlog_root() -> Optional[Path]:
    """Find .tlog directory, searching up from cwd."""
    current = Path.cwd()
    while current != current.parent:
        tlog_path = current / TLOG_DIR
        if tlog_path.is_dir():
            return tlog_path
        current = current.parent
    return None


def require_tlog() -> Path:
    """Get tlog root or exit with error."""
    root = get_tlog_root()
    if not root:
        error_json({"error": "not_initialized", "message": "Not a tlog repository. Run 'tlog init' first."})
    return root


def generate_id() -> str:
    """Generate a short unique ID like tl-a1b2c3d4."""
    now = datetime.now(timezone.utc).isoformat()
    hash_input = f"{now}-{os.urandom(8).hex()}"
    short_hash = hashlib.sha256(hash_input.encode()).hexdigest()[:8]
    return f"tl-{short_hash}"


def now_iso() -> str:
    """Current UTC timestamp in ISO format."""
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def today_str() -> str:
    """Today's date as YYYY-MM-DD."""
    return datetime.now(timezone.utc).strftime("%Y-%m-%d")


def output_json(data: dict | list) -> None:
    """Print JSON to stdout."""
    print(json.dumps(data, indent=2))


def error_json(data: dict) -> None:
    """Print error JSON to stderr and exit."""
    print(json.dumps(data, indent=2), file=sys.stderr)
    sys.exit(1)


def append_event(root: Path, event: dict) -> None:
    """Append an event to today's event file."""
    events_dir = root / EVENTS_DIR
    events_dir.mkdir(exist_ok=True)
    
    event_file = events_dir / f"{today_str()}.jsonl"
    with open(event_file, "a") as f:
        f.write(json.dumps(event, separators=(",", ":")) + "\n")


def load_all_events(root: Path) -> list[dict]:
    """Load all events from all event files, sorted by timestamp."""
    events = []
    events_dir = root / EVENTS_DIR
    
    if not events_dir.exists():
        return events
    
    for event_file in sorted(events_dir.glob("*.jsonl")):
        with open(event_file) as f:
            for line in f:
                line = line.strip()
                if line:
                    try:
                        events.append(json.loads(line))
                    except json.JSONDecodeError:
                        continue
    
    return sorted(events, key=lambda e: e.get("ts", ""))


def compute_state(events: list[dict]) -> dict[str, dict]:
    """
    Replay events to compute current state.
    Returns dict of task_id -> task_state.
    """
    tasks = {}
    
    for event in events:
        task_id = event.get("id")
        event_type = event.get("type")
        
        if event_type == EVENT_CREATE:
            tasks[task_id] = {
                "id": task_id,
                "title": event.get("title", ""),
                "status": event.get("status", "open"),
                "deps": event.get("deps", []),
                "blocks": event.get("blocks", []),
                "created": event.get("ts"),
                "updated": event.get("ts"),
                "labels": event.get("labels", []),
                "notes": event.get("notes", ""),
            }
        
        elif event_type == EVENT_STATUS and task_id in tasks:
            tasks[task_id]["status"] = event.get("status")
            tasks[task_id]["updated"] = event.get("ts")
            if event.get("status") == "done":
                tasks[task_id]["completed"] = event.get("ts")
        
        elif event_type == EVENT_DEP and task_id in tasks:
            dep_id = event.get("dep")
            action = event.get("action", "add")
            if action == "add" and dep_id not in tasks[task_id]["deps"]:
                tasks[task_id]["deps"].append(dep_id)
            elif action == "remove" and dep_id in tasks[task_id]["deps"]:
                tasks[task_id]["deps"].remove(dep_id)
            tasks[task_id]["updated"] = event.get("ts")
        
        elif event_type == EVENT_BLOCK and task_id in tasks:
            block_id = event.get("blocks")
            action = event.get("action", "add")
            if action == "add" and block_id not in tasks[task_id]["blocks"]:
                tasks[task_id]["blocks"].append(block_id)
            elif action == "remove" and block_id in tasks[task_id]["blocks"]:
                tasks[task_id]["blocks"].remove(block_id)
            tasks[task_id]["updated"] = event.get("ts")
        
        elif event_type == EVENT_UPDATE and task_id in tasks:
            if "title" in event:
                tasks[task_id]["title"] = event["title"]
            if "notes" in event:
                tasks[task_id]["notes"] = event["notes"]
            if "labels" in event:
                tasks[task_id]["labels"] = event["labels"]
            tasks[task_id]["updated"] = event.get("ts")
    
    return tasks


def get_ready_tasks(tasks: dict[str, dict]) -> list[dict]:
    """Get tasks that are ready to work on (open, all deps done)."""
    ready = []
    
    for task in tasks.values():
        if task["status"] != "open":
            continue
        
        # Check if all dependencies are done
        deps_satisfied = True
        for dep_id in task["deps"]:
            dep_task = tasks.get(dep_id)
            if not dep_task or dep_task["status"] != "done":
                deps_satisfied = False
                break
        
        # Check if blocked by any open task
        blocked = False
        for other in tasks.values():
            if task["id"] in other.get("blocks", []) and other["status"] != "done":
                blocked = True
                break
        
        if deps_satisfied and not blocked:
            ready.append(task)
    
    return ready


def build_dependency_graph(tasks: dict[str, dict]) -> dict:
    """Build a dependency graph representation."""
    nodes = []
    edges = []
    
    for task in tasks.values():
        nodes.append({
            "id": task["id"],
            "title": task["title"],
            "status": task["status"],
        })
        
        for dep_id in task["deps"]:
            edges.append({
                "from": dep_id,
                "to": task["id"],
                "type": "depends_on",
            })
        
        for block_id in task["blocks"]:
            edges.append({
                "from": task["id"],
                "to": block_id,
                "type": "blocks",
            })
    
    return {"nodes": nodes, "edges": edges}


def graph_to_mermaid(graph: dict) -> str:
    """Convert graph to Mermaid diagram format."""
    lines = ["graph TD"]
    
    for node in graph["nodes"]:
        status_icon = "✓" if node["status"] == "done" else "○"
        safe_title = node["title"].replace('"', "'")[:30]
        lines.append(f'    {node["id"]}["{status_icon} {safe_title}"]')
    
    for edge in graph["edges"]:
        if edge["type"] == "depends_on":
            lines.append(f'    {edge["from"]} --> {edge["to"]}')
        else:
            lines.append(f'    {edge["from"]} -.->|blocks| {edge["to"]}')
    
    return "\n".join(lines)


# ============ Commands ============

def cmd_init(args):
    """Initialize a new tlog repository."""
    tlog_path = Path.cwd() / TLOG_DIR
    
    if tlog_path.exists():
        error_json({"error": "already_initialized", "message": f"tlog already initialized at {tlog_path}"})
    
    tlog_path.mkdir()
    (tlog_path / EVENTS_DIR).mkdir()
    
    config = {
        "version": "0.1.0",
        "created": now_iso(),
    }
    
    with open(tlog_path / CONFIG_FILE, "w") as f:
        json.dump(config, f, indent=2)
    
    output_json({
        "status": "initialized",
        "path": str(tlog_path),
        "message": "tlog initialized. Add .tlog/ to git."
    })


def cmd_create(args):
    """Create a new task."""
    root = require_tlog()
    
    task_id = generate_id()
    ts = now_iso()
    
    event = {
        "id": task_id,
        "ts": ts,
        "type": EVENT_CREATE,
        "title": args.title,
        "status": "open",
        "deps": args.dep or [],
        "blocks": args.blocks or [],
    }
    
    if args.label:
        event["labels"] = args.label
    
    if args.notes:
        event["notes"] = args.notes
    
    append_event(root, event)
    
    output_json({
        "id": task_id,
        "title": args.title,
        "status": "open",
        "deps": event["deps"],
        "blocks": event.get("blocks", []),
        "created": ts,
    })


def cmd_done(args):
    """Mark a task as done."""
    root = require_tlog()
    
    # Verify task exists
    events = load_all_events(root)
    tasks = compute_state(events)
    
    if args.id not in tasks:
        error_json({"error": "not_found", "message": f"Task {args.id} not found"})
    
    ts = now_iso()
    event = {
        "id": args.id,
        "ts": ts,
        "type": EVENT_STATUS,
        "status": "done",
    }
    
    append_event(root, event)
    
    output_json({
        "id": args.id,
        "status": "done",
        "completed": ts,
    })


def cmd_reopen(args):
    """Reopen a completed task."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    
    if args.id not in tasks:
        error_json({"error": "not_found", "message": f"Task {args.id} not found"})
    
    ts = now_iso()
    event = {
        "id": args.id,
        "ts": ts,
        "type": EVENT_STATUS,
        "status": "open",
    }
    
    append_event(root, event)
    
    output_json({
        "id": args.id,
        "status": "open",
        "reopened": ts,
    })


def cmd_dep(args):
    """Add or remove a dependency."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    
    if args.id not in tasks:
        error_json({"error": "not_found", "message": f"Task {args.id} not found"})
    
    if args.on not in tasks:
        error_json({"error": "not_found", "message": f"Dependency task {args.on} not found"})
    
    ts = now_iso()
    event = {
        "id": args.id,
        "ts": ts,
        "type": EVENT_DEP,
        "dep": args.on,
        "action": "remove" if args.remove else "add",
    }
    
    append_event(root, event)
    
    output_json({
        "id": args.id,
        "dependency": args.on,
        "action": event["action"],
        "updated": ts,
    })


def cmd_block(args):
    """Mark a task as blocking another task."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    
    if args.id not in tasks:
        error_json({"error": "not_found", "message": f"Task {args.id} not found"})
    
    if args.blocks not in tasks:
        error_json({"error": "not_found", "message": f"Blocked task {args.blocks} not found"})
    
    ts = now_iso()
    event = {
        "id": args.id,
        "ts": ts,
        "type": EVENT_BLOCK,
        "blocks": args.blocks,
        "action": "remove" if args.remove else "add",
    }
    
    append_event(root, event)
    
    output_json({
        "id": args.id,
        "blocks": args.blocks,
        "action": event["action"],
        "updated": ts,
    })


def cmd_list(args):
    """List tasks."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    
    result = []
    for task in tasks.values():
        if args.status and args.status != "all" and task["status"] != args.status:
            continue
        result.append(task)
    
    # Sort by created date, newest first
    result.sort(key=lambda t: t.get("created", ""), reverse=True)
    
    output_json({"tasks": result, "count": len(result)})


def cmd_show(args):
    """Show details of a specific task."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    
    if args.id not in tasks:
        error_json({"error": "not_found", "message": f"Task {args.id} not found"})
    
    task = tasks[args.id]
    
    # Add computed fields
    task["deps_status"] = []
    for dep_id in task["deps"]:
        dep = tasks.get(dep_id)
        if dep:
            task["deps_status"].append({"id": dep_id, "title": dep["title"], "status": dep["status"]})
    
    task["blocked_by"] = []
    for other in tasks.values():
        if task["id"] in other.get("blocks", []) and other["status"] != "done":
            task["blocked_by"].append({"id": other["id"], "title": other["title"]})
    
    output_json(task)


def cmd_ready(args):
    """List tasks ready to work on."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    ready = get_ready_tasks(tasks)
    
    # Sort by created date
    ready.sort(key=lambda t: t.get("created", ""))
    
    output_json({"tasks": ready, "count": len(ready)})


def cmd_graph(args):
    """Show dependency graph."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    
    # Filter to open tasks only unless --all
    if not args.all:
        tasks = {k: v for k, v in tasks.items() if v["status"] == "open"}
    
    graph = build_dependency_graph(tasks)
    
    if args.format == "mermaid":
        print(graph_to_mermaid(graph))
    else:
        output_json(graph)


def cmd_prime(args):
    """Output context for AI agent injection."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    ready = get_ready_tasks(tasks)
    
    open_tasks = [t for t in tasks.values() if t["status"] == "open"]
    done_recent = [t for t in tasks.values() if t["status"] == "done"]
    done_recent.sort(key=lambda t: t.get("completed", ""), reverse=True)
    done_recent = done_recent[:5]  # Last 5 completed
    
    prime_data = {
        "summary": {
            "total_open": len(open_tasks),
            "total_done": len([t for t in tasks.values() if t["status"] == "done"]),
            "ready_count": len(ready),
        },
        "ready_tasks": [{"id": t["id"], "title": t["title"]} for t in ready[:10]],
        "recent_completed": [{"id": t["id"], "title": t["title"], "completed": t.get("completed")} for t in done_recent],
        "blocked_tasks": [],
    }
    
    # Find blocked tasks
    for task in open_tasks:
        deps_pending = [d for d in task["deps"] if tasks.get(d, {}).get("status") != "done"]
        if deps_pending:
            prime_data["blocked_tasks"].append({
                "id": task["id"],
                "title": task["title"],
                "waiting_on": deps_pending[:3],
            })
    
    prime_data["blocked_tasks"] = prime_data["blocked_tasks"][:5]
    
    output_json(prime_data)


def cmd_sync(args):
    """Git add and commit .tlog directory."""
    root = require_tlog()
    
    try:
        # Git add
        subprocess.run(["git", "add", str(root)], check=True, capture_output=True)
        
        # Git commit
        msg = args.message or "tlog: sync tasks"
        result = subprocess.run(
            ["git", "commit", "-m", msg, "--", str(root)],
            capture_output=True,
            text=True
        )
        
        if result.returncode == 0:
            output_json({"status": "committed", "message": msg})
        elif "nothing to commit" in result.stdout or "nothing to commit" in result.stderr:
            output_json({"status": "clean", "message": "Nothing to commit"})
        else:
            error_json({"error": "git_error", "message": result.stderr})
            
    except FileNotFoundError:
        error_json({"error": "git_not_found", "message": "git command not found"})
    except subprocess.CalledProcessError as e:
        error_json({"error": "git_error", "message": str(e)})


def cmd_update(args):
    """Update task title or notes."""
    root = require_tlog()
    
    events = load_all_events(root)
    tasks = compute_state(events)
    
    if args.id not in tasks:
        error_json({"error": "not_found", "message": f"Task {args.id} not found"})
    
    ts = now_iso()
    event = {
        "id": args.id,
        "ts": ts,
        "type": EVENT_UPDATE,
    }
    
    if args.title:
        event["title"] = args.title
    if args.notes:
        event["notes"] = args.notes
    if args.label:
        event["labels"] = args.label
    
    if len(event) <= 3:  # Only id, ts, type
        error_json({"error": "no_changes", "message": "No changes specified"})
    
    append_event(root, event)
    
    output_json({
        "id": args.id,
        "updated": ts,
        "changes": {k: v for k, v in event.items() if k not in ("id", "ts", "type")},
    })


def main():
    parser = argparse.ArgumentParser(
        prog="tlog",
        description="Append-only task log for AI coding agents",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)
    
    # init
    subparsers.add_parser("init", help="Initialize tlog repository")
    
    # create
    p_create = subparsers.add_parser("create", help="Create a new task")
    p_create.add_argument("title", help="Task title")
    p_create.add_argument("--dep", "-d", action="append", help="Dependency task ID (can repeat)")
    p_create.add_argument("--blocks", "-b", action="append", help="Task ID this blocks (can repeat)")
    p_create.add_argument("--label", "-l", action="append", help="Label (can repeat)")
    p_create.add_argument("--notes", "-n", help="Task notes")
    
    # done
    p_done = subparsers.add_parser("done", help="Mark task as done")
    p_done.add_argument("id", help="Task ID")
    
    # reopen
    p_reopen = subparsers.add_parser("reopen", help="Reopen a completed task")
    p_reopen.add_argument("id", help="Task ID")
    
    # dep
    p_dep = subparsers.add_parser("dep", help="Add/remove dependency")
    p_dep.add_argument("id", help="Task ID")
    p_dep.add_argument("--on", required=True, help="Dependency task ID")
    p_dep.add_argument("--remove", "-r", action="store_true", help="Remove dependency")
    
    # block
    p_block = subparsers.add_parser("block", help="Mark task as blocking another")
    p_block.add_argument("id", help="Blocking task ID")
    p_block.add_argument("--blocks", required=True, help="Task being blocked")
    p_block.add_argument("--remove", "-r", action="store_true", help="Remove block")
    
    # list
    p_list = subparsers.add_parser("list", help="List tasks")
    p_list.add_argument("--status", "-s", choices=["open", "done", "all"], default="open", help="Filter by status")
    
    # show
    p_show = subparsers.add_parser("show", help="Show task details")
    p_show.add_argument("id", help="Task ID")
    
    # ready
    subparsers.add_parser("ready", help="List tasks ready to work on")
    
    # graph
    p_graph = subparsers.add_parser("graph", help="Show dependency graph")
    p_graph.add_argument("--format", "-f", choices=["json", "mermaid"], default="json", help="Output format")
    p_graph.add_argument("--all", "-a", action="store_true", help="Include done tasks")
    
    # prime
    subparsers.add_parser("prime", help="Output context for AI agents")
    
    # sync
    p_sync = subparsers.add_parser("sync", help="Git commit .tlog/")
    p_sync.add_argument("--message", "-m", help="Commit message")
    
    # update
    p_update = subparsers.add_parser("update", help="Update task")
    p_update.add_argument("id", help="Task ID")
    p_update.add_argument("--title", "-t", help="New title")
    p_update.add_argument("--notes", "-n", help="New notes")
    p_update.add_argument("--label", "-l", action="append", help="Labels (replaces all)")
    
    args = parser.parse_args()
    
    commands = {
        "init": cmd_init,
        "create": cmd_create,
        "done": cmd_done,
        "reopen": cmd_reopen,
        "dep": cmd_dep,
        "block": cmd_block,
        "list": cmd_list,
        "show": cmd_show,
        "ready": cmd_ready,
        "graph": cmd_graph,
        "prime": cmd_prime,
        "sync": cmd_sync,
        "update": cmd_update,
    }
    
    commands[args.command](args)


if __name__ == "__main__":
    main()
