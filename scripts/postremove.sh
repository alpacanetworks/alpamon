#!/bin/sh

CONF_FILE_PATH="/etc/alpamon/alpamon.conf"
TMP_FILE_PATH="/usr/lib/tmpfiles.d/alpamon.conf"
SVC_FILE_PATH="/lib/systemd/system/alpamon.service"
LOG_FILE_PATH="/var/log/alpamon/alpamon.log"

if [ "$1" = 'purge' ]; then
    rm -f "$CONF_FILE_PATH" || true
    rm -f "$TMP_FILE_PATH" || true
    rm -f "$SVC_FILE_PATH" || true
    rm -f "$LOG_FILE_PATH" || true

    echo "All related configuration, service, and log files have been deleted."
fi