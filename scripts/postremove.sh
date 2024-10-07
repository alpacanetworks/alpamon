#!/bin/sh

TMP_FILE_PATH="/usr/lib/tmpfiles.d/alpamon.conf"
SVC_FILE_PATH="/lib/systemd/system/alpamon.service"
LOG_FILE_PATH="/var/log/alpamon/alpamon.log"

main() {
  clean_systemd
  clean_files
  clean_directories
  echo "Alpamon has been removed successfully!"
}

clean_systemd() {
  echo "Uninstalling systemd service for Alpamon..."

  systemctl stop alpamon.service || true
  systemctl disable alpamon.service || true
  systemctl daemon-reload || true
}

clean_files() {
  echo "Removing configuration files..."

  rm -f /etc/alpamon/alpamon.conf || true
  rm -f "$TMP_FILE_PATH" || true
  rm -f "$SVC_FILE_PATH" || true
  rm -f "$LOG_FILE_PATH" || true
}

clean_directories() {
  echo "Removing directories..."

  rm -rf /etc/alpamon 2>/dev/null || true
  rm -rf /var/log/alpamon 2>/dev/null || true
}

set -ue
main