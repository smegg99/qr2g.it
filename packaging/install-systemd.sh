#!/usr/bin/env bash
set -euo pipefail

BINARY_NAME="${BINARY_NAME:-qr2git}"
SERVICE_NAME="${SERVICE_NAME:-qr2git}"

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UNIT_TEMPLATE="$APP_DIR/$BINARY_NAME.service.template"
UNIT_PATH="/etc/systemd/system/$SERVICE_NAME.service"

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "run as root, for example: sudo ./install-systemd.sh" >&2
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl not found; this installer is for systemd hosts" >&2
  exit 1
fi

if [[ ! -x "$APP_DIR/$BINARY_NAME" ]]; then
  echo "missing executable: $APP_DIR/$BINARY_NAME" >&2
  exit 1
fi

if [[ ! -f "$UNIT_TEMPLATE" ]]; then
  echo "missing unit template: $UNIT_TEMPLATE" >&2
  exit 1
fi

if [[ ! -f "$APP_DIR/.env" ]]; then
  echo "missing env file: $APP_DIR/.env" >&2
  echo "create it first, for example: cp .env.example .env" >&2
  exit 1
fi

escape_sed() {
  printf '%s' "$1" | sed -e 's/[\/&]/\\&/g'
}

sed \
  -e "s/{{APP_DIR}}/$(escape_sed "$APP_DIR")/g" \
  -e "s/{{BINARY_NAME}}/$(escape_sed "$BINARY_NAME")/g" \
  "$UNIT_TEMPLATE" >"$UNIT_PATH"
chmod 0644 "$UNIT_PATH"

systemctl daemon-reload
systemctl enable "$SERVICE_NAME.service"

cat <<EOF
Installed $SERVICE_NAME service.

Files:
  binary:  $APP_DIR/$BINARY_NAME
  config:  $APP_DIR/config.json
  env:     $APP_DIR/.env
  service: $UNIT_PATH

Start and inspect:
  sudo systemctl start $SERVICE_NAME
  sudo systemctl status $SERVICE_NAME
EOF
