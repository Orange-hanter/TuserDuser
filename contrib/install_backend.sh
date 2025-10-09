#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME=tuserduser-backend
INSTALL_DIR=/var/lib/tuserduser
BIN_SRC=$(pwd)/bin/tuserduser-backend
BIN_DST=/usr/local/bin/${SERVICE_NAME}
SERVICE_FILE=$(pwd)/contrib/systemd/tuserduser-backend.service

if [ "$(id -u)" -ne 0 ]; then
  echo "Please run as root: sudo $0" >&2
  exit 2
fi

echo "Creating user and group 'tuserduser'..."
if ! id -u tuserduser >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin tuserduser
fi

echo "Creating install directory ${INSTALL_DIR}..."
mkdir -p ${INSTALL_DIR}
chown tuserduser:tuserduser ${INSTALL_DIR}

echo "Installing binary to ${BIN_DST}..."
cp "${BIN_SRC}" "${BIN_DST}"
chmod 0755 "${BIN_DST}"

echo "Installing systemd service..."
cp "${SERVICE_FILE}" /etc/systemd/system/
systemctl daemon-reload
systemctl enable ${SERVICE_NAME}

echo "Start service with: systemctl start ${SERVICE_NAME}"
