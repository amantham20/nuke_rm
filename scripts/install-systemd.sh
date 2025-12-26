#!/usr/bin/env bash
# Install nuke systemd service and timer
# Usage: ./install-systemd.sh [--user|--system] [--nuke-path /full/path/to/nuke]

set -euo pipefail

MODE="user"
NUKE_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --system)
      MODE="system"
      shift
      ;;
    --user)
      MODE="user"
      shift
      ;;
    --nuke-path)
      NUKE_PATH="$2"
      shift 2
      ;;
    *)
      echo "Unknown arg: $1"
      exit 1
      ;;
  esac
done

if [[ -n "$NUKE_PATH" ]]; then
  # Replace ExecStart in service file with provided path
  echo "Using provided nuke binary: $NUKE_PATH"
  sed "s|ExecStart=/usr/bin/env nuke --cleanup-trash|ExecStart=$NUKE_PATH --cleanup-trash|" systemd/nuke-cleanup.service > /tmp/nuke-cleanup.service
  SERVICE_SRC="/tmp/nuke-cleanup.service"
else
  SERVICE_SRC="systemd/nuke-cleanup.service"
fi

if [[ "$MODE" == "user" ]]; then
  if ! command -v systemctl >/dev/null 2>&1; then
    echo "systemctl not found. Are you on a systemd-based Linux?"
    exit 1
  fi
  DST_DIR="$HOME/.config/systemd/user"
  mkdir -p "$DST_DIR"
  cp "$SERVICE_SRC" "$DST_DIR/nuke-cleanup.service"
  cp systemd/nuke-cleanup.timer "$DST_DIR/nuke-cleanup.timer"

  echo "Installing user timer in $DST_DIR"
  systemctl --user daemon-reload
  systemctl --user enable --now nuke-cleanup.timer
  echo "Enabled and started nuke-cleanup.timer (user). Use 'systemctl --user status nuke-cleanup.timer' to inspect."
else
  # system mode requires root privileges
  if [[ $(id -u) -ne 0 ]]; then
    echo "System mode requires root. Run with sudo."
    exit 1
  fi
  DST_DIR="/etc/systemd/system"
  cp "$SERVICE_SRC" "$DST_DIR/nuke-cleanup.service"
  cp systemd/nuke-cleanup.timer "$DST_DIR/nuke-cleanup.timer"

  systemctl daemon-reload
  systemctl enable --now nuke-cleanup.timer
  echo "Enabled and started nuke-cleanup.timer (system). Use 'systemctl status nuke-cleanup.timer' to inspect."
fi

# Cleanup temporary file if used
if [[ -n "$NUKE_PATH" && -f /tmp/nuke-cleanup.service ]]; then
  rm /tmp/nuke-cleanup.service
fi

echo "Done. If the nuke binary is not in PATH for the user service, provide --nuke-path /full/path/to/nuke or modify the installed unit file to point to the binary." 
