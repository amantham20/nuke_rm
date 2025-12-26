#!/bin/bash
# Post-installation script for nuke

set -e

echo "nuke has been installed successfully!"
echo ""
echo "Quick start:"
echo "  nuke --help          Show help"
echo "  nuke file.txt        Delete a file (moves to trash)"
echo "  nuke -r directory/   Delete directory recursively"
echo "  nuke --show-trash    Show trash contents"
echo ""
echo "Optional: Enable automatic trash cleanup"
echo "  sudo systemctl enable nuke-cleanup.timer"
echo "  sudo systemctl start nuke-cleanup.timer"
echo ""
echo "Configuration:"
echo "  Copy /etc/nuke/config.example.yaml to ~/.config/nuke/config.yaml"
echo "  and customize as needed."
