#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME=tuserduser-backend
INSTALL_DIR=/var/lib/tuserduser
BIN_DST=/usr/local/bin/${SERVICE_NAME}

if [ "$(id -u)" -ne 0 ]; then
  echo "Please run as root: sudo $0" >&2
  exit 2
fi

echo "Stopping service..."
systemctl stop ${SERVICE_NAME} || true
systemctl disable ${SERVICE_NAME} || true

echo "Removing service file..."
rm -f /etc/systemd/system/${SERVICE_NAME}.service
systemctl daemon-reload

echo "Removing binary ${BIN_DST}..."
rm -f "${BIN_DST}"

echo "Removing install directory ${INSTALL_DIR}..."
rm -rf "${INSTALL_DIR}"

echo "Note: the system user 'tuserduser' has not been removed. Remove manually if desired."
