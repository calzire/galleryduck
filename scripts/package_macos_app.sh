#!/usr/bin/env bash
set -euo pipefail

ARCH="${1:-arm64}"
if [[ "${ARCH}" != "arm64" && "${ARCH}" != "amd64" ]]; then
  echo "usage: $0 <arm64|amd64> [version]"
  exit 1
fi
RAW_VERSION="${2:-local}"
PLIST_VERSION="${RAW_VERSION#v}"

OUT_DIR="dist/macos-${ARCH}"
APP_ROOT="${OUT_DIR}/GalleryDuck.app"
BIN_PATH="${APP_ROOT}/Contents/MacOS/galleryduckd"
LAUNCHER_PATH="${APP_ROOT}/Contents/MacOS/GalleryDuck"

rm -rf "${OUT_DIR}"
mkdir -p "${APP_ROOT}/Contents/MacOS"
mkdir -p "${APP_ROOT}/Contents/Resources"

GOOS=darwin GOARCH="${ARCH}" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "${BIN_PATH}" cmd/api/main.go
chmod +x "${BIN_PATH}"

cat > "${LAUNCHER_PATH}" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN="${SCRIPT_DIR}/galleryduckd"
LOG_DIR="${HOME}/Library/Logs/GalleryDuck"
LOG_FILE="${LOG_DIR}/galleryduck.log"
PID_FILE="${HOME}/Library/Application Support/GalleryDuck/galleryduck.pid"
CONFIG_FILE="${HOME}/.galleryduck/config.json"
DEFAULT_PORT="8787"

mkdir -p "${LOG_DIR}"
mkdir -p "$(dirname "${PID_FILE}")"

get_pid() {
  if [[ -f "${PID_FILE}" ]]; then
    tr -d '\r\n' < "${PID_FILE}" 2>/dev/null || true
    return 0
  fi
  echo ""
}

is_running() {
  local pid
  pid="$(get_pid)"
  if [[ -z "${pid}" ]]; then
    return 1
  fi
  if kill -0 "${pid}" >/dev/null 2>&1; then
    return 0
  fi
  rm -f "${PID_FILE}"
  return 1
}

is_valid_port() {
  local value="$1"
  if [[ ! "${value}" =~ ^[0-9]+$ ]]; then
    return 1
  fi
  if (( value < 1 || value > 65535 )); then
    return 1
  fi
  return 0
}

configured_port() {
  if is_valid_port "${PORT:-}"; then
    echo "${PORT}"
    return
  fi

  if [[ -f "${CONFIG_FILE}" ]]; then
    local file_port
    file_port="$(
      sed -n 's/.*"port"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p' "${CONFIG_FILE}" \
        | head -n 1
    )"
    if is_valid_port "${file_port}"; then
      echo "${file_port}"
      return
    fi
  fi

  echo "${DEFAULT_PORT}"
}

running_port() {
  local pid="$1"
  if [[ -z "${pid}" ]]; then
    return 1
  fi

  local endpoint port
  endpoint="$(lsof -Pan -p "${pid}" -iTCP -sTCP:LISTEN 2>/dev/null | awk 'NR==2 {print $9}')"
  if [[ -z "${endpoint}" ]]; then
    return 1
  fi

  port="${endpoint##*:}"
  port="${port%% *}"
  if ! is_valid_port "${port}"; then
    return 1
  fi

  echo "${port}"
}

effective_port() {
  local pid port
  pid="$(get_pid)"
  if is_running; then
    port="$(running_port "${pid}" || true)"
    if is_valid_port "${port}"; then
      echo "${port}"
      return
    fi
  fi
  configured_port
}

effective_url() {
  local port
  port="$(effective_port)"
  echo "http://localhost:${port}"
}

start_server() {
  if is_running; then
    return 0
  fi
  nohup "${BIN}" >>"${LOG_FILE}" 2>&1 &
  echo "$!" > "${PID_FILE}"
}

stop_server() {
  local pid
  pid="$(get_pid)"
  if [[ -z "${pid}" ]]; then
    rm -f "${PID_FILE}"
    return 0
  fi
  if kill -0 "${pid}" >/dev/null 2>&1; then
    kill "${pid}" >/dev/null 2>&1 || true
    sleep 0.2
    kill -9 "${pid}" >/dev/null 2>&1 || true
  fi
  rm -f "${PID_FILE}"
}

choose_action() {
  local message="$1"
  local buttons_csv="$2"
  local default_button="$3"
  /usr/bin/osascript - "${message}" "${buttons_csv}" "${default_button}" <<'OSA'
on run argv
  set dialogText to item 1 of argv
  set buttonsCSV to item 2 of argv
  set defaultButtonText to item 3 of argv
  set AppleScript's text item delimiters to "|"
  set buttonList to every text item of buttonsCSV
  set reply to display dialog dialogText buttons buttonList default button defaultButtonText with title "GalleryDuck"
  return button returned of reply
end run
OSA
}

choose_action_jxa() {
  local message="$1"
  local buttons_csv="$2"
  local default_button="$3"
  /usr/bin/osascript -l JavaScript - "${message}" "${buttons_csv}" "${default_button}" <<'JXA'
function run(argv) {
  var app = Application.currentApplication();
  app.includeStandardAdditions = true;
  var message = argv[0];
  var buttons = argv[1].split("|");
  var defaultButton = argv[2];
  var result = app.displayDialog(message, {
    buttons: buttons,
    defaultButton: defaultButton,
    withTitle: "GalleryDuck"
  });
  return result.buttonReturned;
}
JXA
}

if is_running; then
  pid="$(get_pid)"
  URL="$(effective_url)"
  CHOICE="$(choose_action "GalleryDuck Server is running.
PID: ${pid}
URL: ${URL}" "Stop Server|Open in Browser|View Logs" "Open in Browser" 2>>"${LOG_FILE}" || true)"
  if [[ -z "${CHOICE}" ]]; then
    CHOICE="$(choose_action_jxa "GalleryDuck Server is running.
PID: ${pid}
URL: ${URL}" "Stop Server|Open in Browser|View Logs" "Open in Browser" 2>>"${LOG_FILE}" || true)"
  fi
else
  URL="$(effective_url)"
  CHOICE="$(choose_action "GalleryDuck Server is not running.
URL: ${URL}" "Start Server|Open in Browser|View Logs" "Start Server" 2>>"${LOG_FILE}" || true)"
  if [[ -z "${CHOICE}" ]]; then
    CHOICE="$(choose_action_jxa "GalleryDuck Server is not running.
URL: ${URL}" "Start Server|Open in Browser|View Logs" "Start Server" 2>>"${LOG_FILE}" || true)"
  fi
fi

if [[ -z "${CHOICE}" ]]; then
  echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ") dialog unavailable; no action taken" >> "${LOG_FILE}"
  exit 1
fi

case "${CHOICE}" in
  "Start Server")
    start_server
    URL="$(effective_url)"
    /usr/bin/open "${URL}" >/dev/null 2>&1 || true
    ;;
  "Stop Server")
    stop_server
    ;;
  "Open in Browser")
    if ! is_running; then
      start_server
    fi
    URL="$(effective_url)"
    /usr/bin/open "${URL}" >/dev/null 2>&1 || true
    ;;
  "View Logs")
    touch "${LOG_FILE}"
    /usr/bin/open "${LOG_FILE}" >/dev/null 2>&1 || true
    ;;
  *)
    ;;
esac

exit 0
SH
chmod +x "${LAUNCHER_PATH}"

cat > "${APP_ROOT}/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CFBundleDisplayName</key>
    <string>GalleryDuck</string>
    <key>CFBundleExecutable</key>
    <string>GalleryDuck</string>
    <key>CFBundleIdentifier</key>
    <string>com.galleryduck.app</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>GalleryDuck</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${PLIST_VERSION}</string>
    <key>CFBundleVersion</key>
    <string>${PLIST_VERSION}</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
  </dict>
</plist>
PLIST

ARCHIVE="${OUT_DIR}/galleryduck_${RAW_VERSION}_darwin_${ARCH}.tar.gz"
tar -C "${OUT_DIR}" -czf "${ARCHIVE}" "GalleryDuck.app"

echo "Built app bundle: ${APP_ROOT}"
echo "Built archive: ${ARCHIVE}"
