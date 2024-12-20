#!/bin/sh

if [ "$1" = 'remove' ]; then
    echo 'Stopping and disabling Alpamon service...'

    if command -v systemctl >/dev/null; then
        systemctl stop alpamon.service || true
        systemctl disable alpamon.service || true
    else
        echo "Systemctl is not available. Skipping service management."
    fi
fi