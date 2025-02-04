#!/bin/bash

ALPAMON_BIN="/usr/local/bin/alpamon"

main() {
  check_root_permission
  check_systemd_status
  check_alpamon_binary
  install_atlas_cli
  install_alpamon
  start_systemd_service
}

check_root_permission() {
  if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run the script as root."
    exit 1
  fi
}

check_systemd_status() {
  if ! command -v systemctl &> /dev/null; then
    echo "Error: systemctl is required but could not be found. Please ensure systemd is installed and systemctl is available."
    exit 1
  fi
}

check_alpamon_binary() {
  if [ ! -f "$ALPAMON_BIN" ]; then
    echo "Error: Alpamon binary not found at $ALPAMON_BIN"
    exit 1
  fi
}

install_atlas_cli() {
  echo "Installing Atlas CLI..."
  curl -sSf https://atlasgo.sh | sh -s -- -y
  if [ $? -ne 0 ]; then
    echo "Error: Failed to install Atlas CLI."
    exit 1
  fi
}

setup_alpamon() {
  "$ALPAMON_BIN" setup
  if [ $? -ne 0 ]; then
    echo "Error: Alpamon setup command failed."
    exit 1
  fi
}

start_systemd_service() {
  echo "Starting systemd service for Alpamon..."

  systemctl daemon-reload || true
  systemctl restart alpamon.service || true
  systemctl enable alpamon.service || true
  systemctl --no-pager status alpamon.service || true

  echo "Alpamon has been installed as a systemd service and will be launched automatically on system boot."
}

set -ue
main