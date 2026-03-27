# ProManNK: Visual Process Orchestrator

## Overview
ProManNK is a low-level, terminal-based Process Orchestrator written in Go. Unlike standard flat-list utilities such as `top` or `ps`, ProManNK visualizes the operating system's workload as a hierarchical process tree. It allows users to interactively navigate parent-child process relationships, monitor real-time resource consumption, and execute system-level signals (such as graceful terminations and cascading forceful kills) directly from the terminal interface.

This project was developed to practically demonstrate core Operating System concepts, including process control blocks, system calls, process state management, and concurrent metric polling.

## Core OS Concepts Demonstrated
* **Process Hierarchies:** Maps the `PID` to `PPID` relationships, visually demonstrating the `fork()` and `exec()` execution flow of the Linux kernel.
* **System Calls & Inter-Process Communication (IPC):** Utilizes the native OS `syscall` interface to dispatch POSIX signals (`SIGTERM`, `SIGKILL`) to specific processes.
* **Process State & Resource Management:** Reads and formats live data from the `/proc` filesystem (via `gopsutil`), exposing scheduling priorities (nice values), CPU utilization, and memory allocation.
* **Concurrency:** Employs Go's native goroutines for non-blocking, asynchronous background polling of system metrics without interrupting the main UI rendering thread.

## Features
* **Interactive Tree Visualization:** Expand and collapse process branches to trace execution lineage from `systemd` (PID 1) downwards.
* **Granular Process Control:** Send individual signals to specific processes to request graceful shutdown or force immediate termination.
* **Cascading Operations:** Recursively traverse the process tree to safely terminate a parent process and all of its subsequent child processes in a single operation.
* **Real-Time Telemetry:** View live CPU% and Memory% utilization alongside process ownership and execution commands.

## Installation and Execution

### Prerequisites
* Go 1.20 or higher
* A Linux-based operating system (or macOS)

### Build Instructions
To compile the application into a standalone binary:
```bash
go build -o promannk main.go
```
### Execution
To view processes, simply run the binary.
Note: To execute system calls (like terminating system-level processes or processes owned by other users), the binary must be executed with elevated privileges.
```bash
# Standard execution (Read-only for other users' processes)
./promannk

# Elevated execution (Required for full signal control)
sudo ./promannk
```

## Keybindings
| __Key__        | __Action__           | __Description__                                                        |
|----------------|----------------------|------------------------------------------------------------------------|
| `↑` / `k`      | Navigate Up          | 	Moves the cursor up the visible process list.                         |
| `↓` / `j`      | Navigate Down        | 	Moves the cursor down the visible process list.                       |
| `Enter`        | Toggle Tree          | Expands or collapses the selected process's children.                  |
| `t`            | Terminate (Graceful) | 	Sends SIGTERM to the selected process.                                |
| `f`            | Force Kill           | 	Sends SIGKILL to the selected process.                                |
| `T`            | Tree Terminate       | 	Recursively sends SIGTERM to the selected process and all children.   |
| `F`            | Tree Force Kill      | 	Recursively sends SIGKILL to the selected process and all children.   |
| `q` / `Ctrl+C` | Quit                 | Safely exits the alternate terminal buffer and closes the application. |

## Technical Stack
- __Language__ : Go
- __UI Framework__: `github.com/charmbracelet/bubbletea` (Elm-architecture TUI framework)
- __Styling__: `github.com/charmbracelet/lipgloss`
- __System Metrics__: `github.com/shirou/gopsutil`
---
Made by [Nirbhik Kumawat](https://github.com/NirbhikKumawat)
