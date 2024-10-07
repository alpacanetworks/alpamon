#!/bin/bash

ALPAMON_BIN="/usr/local/bin/alpamon"

main() {
  check_root_permission
  check_systemd_status
  check_alpamon_binary
  install_alpamon
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

install_alpamon() {
  echo "Running Alpamon install command..."
  "$ALPAMON_BIN" install
  if [ $? -ne 0 ]; then
    echo "Error: Alpamon install command failed."
    exit 1
  fi
  echo "Alpamon has been successfully installed."
}

set -ue
main