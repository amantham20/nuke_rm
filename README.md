# nuke - A Safer, Faster, and More User-Friendly Alternative to rm

`nuke` is a command-line utility written in Go that provides a safer, faster, and more user-friendly alternative to the standard `rm` command. It features soft-delete (trash) functionality, concurrent file deletion, smart filtering, safety checks, and secure file shredding.

## Features

### 🛡️ Safety Features
- **Soft Delete (Trash)**: Files are moved to a trash directory instead of being permanently deleted
- **Protected Paths**: Critical system paths are protected from accidental deletion
- **Dangerous Pattern Detection**: Warns about risky operations like deleting `/` or using wildcards
- **Typed Confirmation**: Requires "yes I am sure" for dangerous operations
- **Countdown Timer**: 5-second countdown for large deletions (Ctrl+C to abort)
- **Confirmation Prompts**: Asks for y/n confirmation before proceeding

### ⚡ Performance
- **Concurrent Deletion**: Multi-threaded file deletion using configurable worker pools
- **Progress Bar**: Visual progress indication for large operations
- **Efficient Scanning**: Fast file system traversal

### 🎯 Smart Filtering
- **Time-based**: `--older-than=30d`, `--newer-than=24h`
- **Size-based**: `--size=+100M` (larger than), `--size=-1G` (smaller than)
- **Pattern-based**: `--exclude=*.cfg`, `--include=*.log`
- **Regex Support**: `--regex=".*\.tmp$"`

### 🔒 Security
- **Secure Shredding**: Overwrites files with random data before deletion (`--shred`)
- **Multiple Overwrite Passes**: DoD-standard 3-pass overwrite

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/nuke.git
cd nuke

# Build
go build -o nuke .

# Install (optional)
sudo mv nuke /usr/local/bin/
```

### Requirements
- Go 1.21 or later

## Usage

### Basic Commands

```bash
# Delete a single file
nuke file.txt

# Delete directory recursively
nuke -r directory/

# Delete with force (skip confirmation)
nuke -f file.txt

# Preview what would be deleted (dry run)
nuke --dry-run *.log
```

### Filtering

```bash
# Delete files older than 30 days
nuke -r --older-than=30d logs/

# Delete files larger than 100MB
nuke -r --size=+100M downloads/

# Delete all except .cfg files
nuke -r --exclude=*.cfg config/

# Delete files matching regex
nuke -r --regex=".*\.tmp$" /tmp/
```

### Trash Operations

```bash
# Show what's in the trash
nuke --show-trash

# Restore a file from trash
nuke --restore=file.txt

# Auto-cleanup trash based on retention policy
nuke --cleanup-trash

# Empty the trash permanently
nuke --empty-trash
```

### Secure Deletion

```bash
# Securely shred a sensitive file (bypasses trash)
nuke --shred secret.txt
```

### Interactive Mode

```bash
# Ask for confirmation for each file
nuke -i -r directory/
# Options: y (yes), n (no), a (all), q (quit)
```

## Options

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help message |
| `-r, --recursive` | Delete directories recursively |
| `-f, --force` | Skip confirmation prompts |
| `-i, --interactive` | Ask for confirmation for each file |
| `-v, --verbose` | Show detailed output |
| `--dry-run` | Preview deletion without modifying files |
| `--shred` | Securely overwrite files before deletion |
| `--no-countdown` | Skip the countdown timer |
| `--empty-trash` | Permanently delete all files in trash |
| `--restore=<file>` | Restore a file from trash |
| `--older-than=<dur>` | Filter by age (e.g., 30d, 24h, 1w) |
| `--newer-than=<dur>` | Filter by age |
| `--size=<size>` | Filter by size (+100M for >100MB, -1G for <1GB) |
| `--exclude=<pattern>` | Exclude files matching glob pattern |
| `--include=<pattern>` | Include only files matching glob pattern |
| `--regex=<pattern>` | Match files using regex pattern |
| `--workers=<n>` | Number of concurrent workers (default: 8) |

## Configuration

You can customize protected paths by creating a config file at `~/.config/nuke/config.yaml`:

```yaml
# Additional paths to protect from deletion
protected_paths:
  - "~/important_project"
  - "/data/backups"
  - "*.critical"
```

### Default Protected Paths

The following paths are protected by default:
- `/`, `/bin`, `/sbin`, `/usr`, `/etc`, `/var`, `/lib`, `/boot`
- `/sys`, `/proc`, `/dev`, `/System`, `/Library` (macOS)
- `~/.ssh`, `~/.gnupg`, `~/.config`
- `.git/` directories

## Trash Location

Deleted files are stored in `~/.nuke-trash/`:
- `~/.nuke-trash/files/` - Actual files
- `~/.nuke-trash/meta/` - Metadata for restoration

### Automatic Trash Cleanup

The trash is **never automatically deleted on its own**. Files persist in trash indefinitely until you explicitly manage them:

1. **Manual empty**: `nuke --empty-trash` - Permanently deletes ALL trash
2. **Auto-cleanup**: `nuke --cleanup-trash` - Removes files older than the retention period or enforces size limits
3. **Restore**: `nuke --restore=<filename>` - Recover deleted files before they're cleaned up

### Trash Retention Policy

Configure retention in `~/.config/nuke/config.yaml`:

```yaml
# How long to keep files in trash before auto-delete (default: 30 days)
trash_retention_days: 30

# Maximum trash directory size in MB (default: 5000 MB = 5 GB)
trash_max_size_mb: 5000

# Enable automatic cleanup (default: true)
auto_cleanup_enabled: true
```

When `nuke --cleanup-trash` runs:
1. Files older than `trash_retention_days` are removed
2. If trash exceeds `trash_max_size_mb`, oldest files are removed first until within limit
3. Files younger than the retention period are kept (unless size limit forces removal)

## Examples

### Delete old log files
```bash
nuke -r --older-than=7d --include="*.log" /var/log/myapp/
```

### Clean up large temporary files
```bash
nuke -r --size=+50M --include="*.tmp" /tmp/
```

## Systemd auto-cleanup (Linux)

If you want `nuke` to automatically run trash cleanup on a schedule using systemd timers, the repository includes a service and timer unit and a small installer script.

Files added:

- `systemd/nuke-cleanup.service` — oneshot service that runs `nuke --cleanup-trash`
- `systemd/nuke-cleanup.timer` — timer unit (defaults to `OnCalendar=daily`)
- `scripts/install-systemd.sh` — helper script to install and enable the timer (supports `--user` and `--system` modes and `--nuke-path` to set binary path)

Install (user mode, recommended):

```bash
# Make the installer executable
chmod +x scripts/install-systemd.sh
# Install into the current user's systemd user units
./scripts/install-systemd.sh --user --nuke-path /usr/local/bin/nuke

# Inspect status
systemctl --user status nuke-cleanup.timer
```

Install system-wide (requires root):

```bash
sudo ./scripts/install-systemd.sh --system --nuke-path /usr/local/bin/nuke
systemctl status nuke-cleanup.timer
```

Notes:
- Systemd timers only apply on Linux systems with systemd. On macOS use `launchd` or a cron job instead.
- If the `nuke` binary is not in the PATH of the systemd service, pass `--nuke-path` or edit the installed service file to point to the full path of the binary.

## launchd auto-cleanup (macOS)

For macOS you can use `launchd` to schedule automatic cleanup. This repository includes a sample plist and an installer script that installs the plist into `~/Library/LaunchAgents` and loads it for the current user.

Files added:

- `launchd/com.nuke.nuke-cleanup.plist` — sample plist that runs `nuke --cleanup-trash` daily at 02:00 by default
- `scripts/install-launchd.sh` — helper script to install and load the plist (supports `--nuke-path` and schedule `--hour`/`--minute`)

Install (user mode, recommended):

```bash
# Make installer executable
chmod +x scripts/install-launchd.sh
# Install into current user's LaunchAgents and load
./scripts/install-launchd.sh --nuke-path /usr/local/bin/nuke --hour 2 --minute 0

# Check the job
launchctl list | grep com.nuke.cleanup
```

Uninstall (user):

```bash
# Unload and remove
launchctl unload ~/Library/LaunchAgents/com.nuke.nuke-cleanup.plist 2>/dev/null || true
rm ~/Library/LaunchAgents/com.nuke.nuke-cleanup.plist
```

Notes:
- `launchd` runs on macOS only. If `launchctl` load fails, run the installer manually or inspect the plist.
- If the `nuke` binary isn't in `/usr/local/bin`, pass the full path with `--nuke-path` to the installer, or edit the plist directly.

(earlier README content retained)

### Secure deletion of sensitive data
```bash
nuke --shred -r sensitive_data/
```

### Preview and confirm
```bash
# First, preview
nuke --dry-run -r old_project/

# Then, delete interactively
nuke -i -r old_project/
```

## Safety Comparison with rm

| Feature | rm | nuke |
|---------|-------|------|
| Undo/Restore | ❌ | ✅ |
| Progress Bar | ❌ | ✅ |
| Dangerous Path Protection | ❌ | ✅ |
| Countdown Timer | ❌ | ✅ |
| Time/Size Filtering | ❌ | ✅ |
| Secure Shredding | ❌ | ✅ |
| Concurrent Deletion | ❌ | ✅ |
| Dry Run Mode | ❌ | ✅ |
| Interactive Mode | Partial | ✅ |

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
