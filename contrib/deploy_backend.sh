#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<EOF
Usage: $0 [options] user@host

Deploy the built backend binary and systemd unit to a remote Ubuntu host,
create the service account and install+enable the systemd unit.

Options:
  -p PORT      SSH port (default: 22)
  -i IDENTITY  SSH private key to use (optional)
  -b BINARY    Path to local binary to deploy (default: bin/tuserduser-backend)
  -s SERVICE   Path to local systemd unit (default: contrib/systemd/tuserduser-backend.service)
  -h           Show this help

Example:
  ./contrib/deploy_backend.sh -p 2222 -i ~/.ssh/id_rsa ubuntu@1.2.3.4

Note: this script uses sudo on the remote host; ensure the remote user can run sudo.
EOF
}

SSH_PORT=22
IDENTITY=""
# By default prefer the linux/amd64 cross-built binary if present
DEFAULT_LINUX_BIN="bin/tuserduser-backend-linux-amd64"
DEFAULT_LOCAL_BIN="bin/tuserduser-backend"
BINARY=""
SERVICE="contrib/systemd/tuserduser-backend.service"
SUPPORT_DIR="Support"

while getopts ":p:i:b:s:h" opt; do
  case ${opt} in
    p ) SSH_PORT=${OPTARG} ;;
    i ) IDENTITY=${OPTARG} ;;
    b ) BINARY=${OPTARG} ;;
    s ) SERVICE=${OPTARG} ;;
    h ) usage; exit 0 ;;
    \? ) echo "Invalid option: -$OPTARG" >&2; usage; exit 1 ;;
  esac
done
shift $((OPTIND -1))

if [ $# -ne 1 ]; then
  usage; exit 1
fi

REMOTE=$1

if [ -z "$BINARY" ]; then
  if [ -f "$DEFAULT_LINUX_BIN" ]; then
    BINARY="$DEFAULT_LINUX_BIN"
  else
    BINARY="$DEFAULT_LOCAL_BIN"
  fi
fi

if [ ! -f "$BINARY" ]; then
  echo "Binary not found: $BINARY" >&2
  exit 2
fi
if [ ! -f "$SERVICE" ]; then
  echo "Service file not found: $SERVICE" >&2
  exit 2
fi

if [ ! -d "$SUPPORT_DIR" ]; then
  echo "Warning: support directory '$SUPPORT_DIR' not found in repo; skipping static deploy." >&2
fi

SSH_OPTS=( -p "$SSH_PORT" )
SCP_OPTS=( -P "$SSH_PORT" )
if [ -n "$IDENTITY" ]; then
  SSH_OPTS+=( -i "$IDENTITY" )
  SCP_OPTS+=( -i "$IDENTITY" )
fi

echo "Uploading binary to ${REMOTE}:/tmp/"
scp ${SCP_OPTS[@]} "$BINARY" "$REMOTE":/tmp/tuserduser-backend

echo "Uploading service unit to ${REMOTE}:/tmp/"
scp ${SCP_OPTS[@]} "$SERVICE" "$REMOTE":/tmp/tuserduser-backend.service

if [ -d "$SUPPORT_DIR" ]; then
  echo "Uploading Support directory to ${REMOTE}:/tmp/Support"
  scp -r ${SCP_OPTS[@]} "$SUPPORT_DIR" "$REMOTE":/tmp/Support
fi

echo "Installing on remote host ${REMOTE}..."
ssh ${SSH_OPTS[@]} "$REMOTE" bash -e <<'REMOTE_SCRIPT'
set -euo pipefail
SERVICE_NAME=tuserduser-backend
BIN_DST=/usr/local/bin/${SERVICE_NAME}
INSTALL_DIR=/var/lib/tuserduser

echo "Creating system user 'tuserduser' if missing..."
if ! id -u tuserduser >/dev/null 2>&1; then
  sudo useradd --system --no-create-home --shell /usr/sbin/nologin tuserduser || true
fi

echo "Creating install directory ${INSTALL_DIR}..."
sudo mkdir -p "${INSTALL_DIR}"
sudo chown tuserduser:tuserduser "${INSTALL_DIR}"

echo "Installing binary to ${BIN_DST}..."
sudo mv /tmp/tuserduser-backend "${BIN_DST}"
sudo chmod 0755 "${BIN_DST}"

echo "Installing systemd unit..."
sudo mv /tmp/tuserduser-backend.service /etc/systemd/system/${SERVICE_NAME}.service
sudo systemctl daemon-reload
sudo systemctl enable --now ${SERVICE_NAME}

# Install Support directory if uploaded
if [ -d /tmp/Support ]; then
  echo "Installing Support files to ${INSTALL_DIR}/Support..."
  sudo rm -rf "${INSTALL_DIR}/Support" || true
  sudo mv /tmp/Support "${INSTALL_DIR}/Support"
  sudo chown -R tuserduser:tuserduser "${INSTALL_DIR}/Support"
fi

echo "Deployment complete. Service status:"
sudo systemctl status ${SERVICE_NAME} --no-pager || true
REMOTE_SCRIPT

echo "Done. Use: ssh ${REMOTE} 'sudo journalctl -u tuserduser-backend -f' to view logs."
