#!/usr/bin/env bash
set -euo pipefail

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "missing required env var: $name" >&2
    exit 1
  fi
}

build_base_url() {
  local host="$1"
  local insecure_http="$2"

  if [[ "$host" == http://* || "$host" == https://* ]]; then
    printf '%s' "${host%/}"
    return
  fi

  if [[ "$insecure_http" == "true" ]]; then
    printf 'http://%s' "${host%/}"
  else
    printf 'https://%s' "${host%/}"
  fi
}

login() {
  echo "POST /logon.cgi"
  curl "${curl_common[@]}" \
    --request POST \
    --header "Content-Type: application/x-www-form-urlencoded" \
    --dump-header "$HEADER_DIR/logon.headers.txt" \
    --data-urlencode "username=$TPLINK_EASYSMART_USERNAME" \
    --data-urlencode "password=$TPLINK_EASYSMART_PASSWORD" \
    --data-urlencode "cpassword=" \
    --data-urlencode "logon=Login" \
    "$BASE_URL/logon.cgi" \
    -o "$BODY_DIR/logon.cgi.html"
}

fetch_get() {
  local path="$1"
  local name="$2"

  echo "GET /$path -> $name"
  curl "${curl_common[@]}" \
    --dump-header "$HEADER_DIR/$name.headers.txt" \
    --trace-ascii "$TRACE_DIR/$name.trace.txt" \
    "$BASE_URL/$path" \
    -o "$BODY_DIR/$name"
}

record_summary() {
  local file="$1"
  local name="$2"

  {
    echo "== $name =="
    if rg -q "var logonInfo|logon.cgi" "$file"; then
      echo "login-page markers detected"
    else
      echo "no login-page markers detected"
    fi
    if rg -q "qos_mode_set\\.cgi" "$file"; then
      echo "contains qos_mode_set.cgi"
    fi
    if rg -q "qos_port_priority_set\\.cgi" "$file"; then
      echo "contains qos_port_priority_set.cgi"
    fi
    if rg -q "qos_bandwidth_set\\.cgi" "$file"; then
      echo "contains qos_bandwidth_set.cgi"
    fi
    if rg -q "qos_storm_set\\.cgi" "$file"; then
      echo "contains qos_storm_set.cgi"
    fi
    if rg -q "QosBandWidthControl|bandWidth|bandwidth" "$file"; then
      echo "contains bandwidth-related markers"
    fi
    if rg -q "QosStormControl|stormControl|storm" "$file"; then
      echo "contains storm-related markers"
    fi
    if rg -q "qosMode|pPri|pTrunk|portNumber" "$file"; then
      echo "contains QoS Basic state markers"
    fi
    if rg -q "bcInfo|igrRate|egrRate" "$file"; then
      echo "contains bandwidth-control state markers"
    fi
    if rg -q "scInfo|stormType|Total Rate|rate" "$file"; then
      echo "contains storm-control state markers"
    fi
    echo
  } >> "$OUT_DIR/summary.txt"
}

write_instructions() {
  cat > "$OUT_DIR/NEXT_STEPS.md" <<'EOF'
# QoS Evidence Capture

This script captured the authenticated QoS pages, request traces for page loads,
and the page-specific JavaScript files that likely build the apply payloads.

Check these files first:

- `bodies/QosBasicRpm.js`
- `bodies/QosBandWidthControlRpm.js`
- `bodies/QosStormControlRpm.js`

If those JS files are not enough to prove the exact write payloads, finish the
evidence set with one real Apply capture per page:

1. Log in to the switch in a browser.
2. Open DevTools Network tab and preserve log.
3. Change one low-risk QoS Basic setting:
   - either one `qosMode` change
   - or one single-port priority change
4. Save/apply the change.
5. Export the request details for the submission to `qos_mode_set.cgi`.
6. Repeat for:
   - bandwidth control
   - storm control

Most useful artifacts:

- full request URL
- method
- form payload
- response body
- updated page HTML after the change

If your browser exports HAR, that is sufficient.
EOF
}

require_env "TPLINK_EASYSMART_HOST"
require_env "TPLINK_EASYSMART_USERNAME"
require_env "TPLINK_EASYSMART_PASSWORD"

INSECURE_HTTP="${TPLINK_EASYSMART_INSECURE_HTTP:-true}"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="${TPLINK_EASYSMART_OUTPUT_DIR:-./qos-capture-$STAMP}"
COOKIE_JAR="$OUT_DIR/cookies.txt"
TRACE_DIR="$OUT_DIR/traces"
BODY_DIR="$OUT_DIR/bodies"
HEADER_DIR="$OUT_DIR/headers"

mkdir -p "$TRACE_DIR" "$BODY_DIR" "$HEADER_DIR"

BASE_URL="$(build_base_url "$TPLINK_EASYSMART_HOST" "$INSECURE_HTTP")"

curl_common=(
  --silent
  --show-error
  --location
  --http1.1
  --cookie "$COOKIE_JAR"
  --cookie-jar "$COOKIE_JAR"
  --user-agent "Mozilla/5.0 (Macintosh; Intel Mac OS X) QoS Capture Script"
)

{
  echo "base_url=$BASE_URL"
  echo "host_env=$TPLINK_EASYSMART_HOST"
  echo "captured_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} > "$OUT_DIR/manifest.txt"

login

fetch_get "" "root.html"
fetch_get "Menu.htm" "Menu.htm"
fetch_get "QosBasicRpm.htm" "QosBasicRpm.htm"
fetch_get "QosBandWidthControlRpm.htm" "QosBandWidthControlRpm.htm"
fetch_get "QosStormControlRpm.htm" "QosStormControlRpm.htm"
fetch_get "QosBasicRpm.js" "QosBasicRpm.js"
fetch_get "QosBandWidthControlRpm.js" "QosBandWidthControlRpm.js"
fetch_get "QosStormControlRpm.js" "QosStormControlRpm.js"

: > "$OUT_DIR/summary.txt"
record_summary "$BODY_DIR/logon.cgi.html" "logon.cgi.html"
record_summary "$BODY_DIR/QosBasicRpm.htm" "QosBasicRpm.htm"
record_summary "$BODY_DIR/QosBandWidthControlRpm.htm" "QosBandWidthControlRpm.htm"
record_summary "$BODY_DIR/QosStormControlRpm.htm" "QosStormControlRpm.htm"
record_summary "$BODY_DIR/QosBasicRpm.js" "QosBasicRpm.js"
record_summary "$BODY_DIR/QosBandWidthControlRpm.js" "QosBandWidthControlRpm.js"
record_summary "$BODY_DIR/QosStormControlRpm.js" "QosStormControlRpm.js"

write_instructions

cat <<EOF
Capture complete.

Output directory: $OUT_DIR

Important files:
  $OUT_DIR/manifest.txt
  $OUT_DIR/summary.txt
  $OUT_DIR/NEXT_STEPS.md
  $BODY_DIR/QosBasicRpm.htm
  $BODY_DIR/QosBandWidthControlRpm.htm
  $BODY_DIR/QosStormControlRpm.htm
  $BODY_DIR/QosBasicRpm.js
  $BODY_DIR/QosBandWidthControlRpm.js
  $BODY_DIR/QosStormControlRpm.js
  $TRACE_DIR/QosBasicRpm.htm.trace.txt
  $TRACE_DIR/QosBandWidthControlRpm.htm.trace.txt
  $TRACE_DIR/QosStormControlRpm.htm.trace.txt
EOF
