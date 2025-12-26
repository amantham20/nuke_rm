#!/usr/bin/env bash
# Install nuke launchd plist for current user
# Usage: ./install-launchd.sh [--nuke-path /full/path/to/nuke] [--hour 2 --minute 0]

set -euo pipefail

NUKE_PATH="/usr/local/bin/nuke"
HOUR=2
MINUTE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --nuke-path)
      NUKE_PATH="$2"
      shift 2
      ;;
    --hour)
      HOUR="$2"
      shift 2
      ;;
    --minute)
      MINUTE="$2"
      shift 2
      ;;
    --help)
      echo "Usage: $0 [--nuke-path /full/path/to/nuke] [--hour 2 --minute 0]"
      exit 0
      ;;
    *)
      echo "Unknown arg: $1"
      exit 1
      ;;
  esac
done

SRC_PLIST="launchd/com.nuke.nuke-cleanup.plist"
DST_DIR="$HOME/Library/LaunchAgents"
DST_PLIST="$DST_DIR/com.nuke.nuke-cleanup.plist"

mkdir -p "$DST_DIR"

# Generate a clean plist with the correct binary path and schedule
cat > "$DST_PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.nuke.cleanup</string>

    <key>ProgramArguments</key>
    <array>
        <string>${NUKE_PATH}</string>
        <string>--cleanup-trash</string>
    </array>

    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>${HOUR}</integer>
        <key>Minute</key>
        <integer>${MINUTE}</integer>
    </dict>

    <key>StandardOutPath</key>
    <string>/tmp/nuke-cleanup.out.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/nuke-cleanup.err.log</string>

    <key>RunAtLoad</key>
    <false/>
</dict>
</plist>
EOF

# Load the job (unload first if already loaded)
launchctl unload "$DST_PLIST" 2>/dev/null || true
launchctl load -w "$DST_PLIST"

echo "Installed and loaded $DST_PLIST"

echo "If the nuke binary is not executable or in a restricted path, pass --nuke-path /full/path/to/nuke when running the installer."
