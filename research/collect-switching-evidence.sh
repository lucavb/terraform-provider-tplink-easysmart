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
  local http_code
  local curl_exit

  set +e
  http_code="$(
    curl "${curl_common[@]}" \
      --request POST \
      --header "Content-Type: application/x-www-form-urlencoded" \
      --dump-header "$HEADER_DIR/logon.headers.txt" \
      --trace-ascii "$TRACE_DIR/logon.trace.txt" \
      --data-urlencode "username=$TPLINK_EASYSMART_USERNAME" \
      --data-urlencode "password=$TPLINK_EASYSMART_PASSWORD" \
      --data-urlencode "cpassword=" \
      --data-urlencode "logon=Login" \
      "$BASE_URL/logon.cgi" \
      --output "$BODY_DIR/logon.cgi.html" \
      --write-out '%{http_code}'
  )"
  curl_exit=$?
  set -e

  {
    echo "http_status=$http_code"
    echo "curl_exit=$curl_exit"
  } > "$META_DIR/logon.cgi.html.status.txt"

  if [[ "$curl_exit" -ne 0 ]]; then
    echo "login failed (curl_exit=$curl_exit, http_status=$http_code)" >&2
    exit 1
  fi
}

fetch_get() {
  local path="$1"
  local name="$2"

  echo "GET /$path -> $name"
  local http_code
  local curl_exit

  set +e
  http_code="$(
    curl "${curl_common[@]}" \
      --dump-header "$HEADER_DIR/$name.headers.txt" \
      --trace-ascii "$TRACE_DIR/$name.trace.txt" \
      "$BASE_URL/$path" \
      --output "$BODY_DIR/$name" \
      --write-out '%{http_code}'
  )"
  curl_exit=$?
  set -e

  {
    echo "http_status=$http_code"
    echo "curl_exit=$curl_exit"
  } > "$META_DIR/$name.status.txt"

  if [[ "$curl_exit" -ne 0 ]]; then
    echo "probe failed for /$path (curl_exit=$curl_exit, http_status=$http_code)" >&2
  fi
}

merge_unique_matches() {
  local pattern="$1"
  local target="$2"
  shift 2

  local tmp
  tmp="$(mktemp)"

  if [[ -f "$target" ]]; then
    cat "$target" >> "$tmp"
  fi

  local source
  for source in "$@"; do
    if [[ -f "$source" ]]; then
      rg -o "$pattern" "$source" >> "$tmp" || true
    fi
  done

  if [[ -s "$tmp" ]]; then
    sort -u "$tmp" > "$target"
  fi

  rm -f "$tmp"
}

seed_targets() {
  cat > "$DISCOVERY_DIR/candidate-pages.txt" <<'EOF'
PortSettingRpm.htm
PortMirrorRpm.htm
PortTrunkRpm.htm
IgmpSnoopingRpm.htm
EOF
}

discover_from_menu() {
  local tmp
  tmp="$(mktemp)"

  if [[ -f "$BODY_DIR/menuList.js" ]]; then
    rg -o 'PortSettingRpm|PortMirrorRpm|PortTrunkRpm|IgmpSnoopingRpm' "$BODY_DIR/menuList.js" \
      | sed 's/$/.htm/' > "$tmp" || true
  fi

  if [[ -s "$tmp" ]]; then
    sort -u "$tmp" > "$DISCOVERY_DIR/discovered-pages.txt"
    cat "$DISCOVERY_DIR/discovered-pages.txt" >> "$DISCOVERY_DIR/candidate-pages.txt"
    sort -u "$DISCOVERY_DIR/candidate-pages.txt" -o "$DISCOVERY_DIR/candidate-pages.txt"
  fi

  rm -f "$tmp"
}

fetch_candidate_pages() {
  local page
  while IFS= read -r page; do
    [[ -z "$page" ]] && continue
    fetch_get "$page" "$page"
  done < "$DISCOVERY_DIR/candidate-pages.txt"
}

discover_from_pages() {
  local page
  while IFS= read -r page; do
    [[ -z "$page" ]] && continue
    merge_unique_matches '[A-Za-z0-9_/-]+\.js' "$DISCOVERY_DIR/discovered-scripts.txt" "$BODY_DIR/$page"
    merge_unique_matches '[A-Za-z0-9_/-]+HelpRpm\.htm' "$DISCOVERY_DIR/discovered-help-pages.txt" "$BODY_DIR/$page"
    merge_unique_matches '[A-Za-z0-9_/-]+\.cgi' "$DISCOVERY_DIR/discovered-cgis.txt" "$BODY_DIR/$page"
  done < "$DISCOVERY_DIR/candidate-pages.txt"

  if [[ -f "$DISCOVERY_DIR/discovered-help-pages.txt" ]]; then
    cat "$DISCOVERY_DIR/discovered-help-pages.txt" >> "$DISCOVERY_DIR/candidate-pages.txt"
    sort -u "$DISCOVERY_DIR/candidate-pages.txt" -o "$DISCOVERY_DIR/candidate-pages.txt"
  fi

  if [[ -f "$DISCOVERY_DIR/discovered-scripts.txt" ]]; then
    sort -u "$DISCOVERY_DIR/discovered-scripts.txt" -o "$DISCOVERY_DIR/discovered-scripts.txt"
  fi
}

fetch_discovered_scripts() {
  local script_path
  while IFS= read -r script_path; do
    [[ -z "$script_path" ]] && continue
    fetch_get "$script_path" "$script_path"
  done < "$DISCOVERY_DIR/discovered-scripts.txt"
}

record_summary() {
  local file="$1"
  local name="$2"
  local status_file="$3"
  local http_status="unknown"
  local curl_exit="unknown"

  if [[ -f "$status_file" ]]; then
    http_status="$(rg '^http_status=' "$status_file" | sed 's/^http_status=//')"
    curl_exit="$(rg '^curl_exit=' "$status_file" | sed 's/^curl_exit=//')"
  fi

  {
    echo "== $name =="
    echo "http_status=$http_status"
    echo "curl_exit=$curl_exit"
    if rg -q 'var logonInfo|logon\.cgi|name=logon' "$file"; then
      echo "login-page markers detected"
    else
      echo "no login-page markers detected"
    fi
    if rg -q 'IGMP|igmp|Snoop|snoop' "$file"; then
      echo "contains IGMP-related markers"
    fi
    if rg -q 'LAG|Lag|lagIds|lagMbrs|trunk_info|pTrunk|Trunk|trunk' "$file"; then
      echo "contains LAG or trunk-related markers"
    fi
    if rg -q 'Port Setting|PortSetting|port_setting\.cgi|spd_cfg|fc_cfg' "$file"; then
      echo "contains port-setting markers"
    fi
    if rg -q 'Rpm\.htm' "$file"; then
      echo "contains Rpm page references"
    fi
    if rg -q '\.cgi' "$file"; then
      echo "contains CGI handler references"
      rg -o '[A-Za-z0-9_/-]+\.cgi' "$file" | sort -u | sed 's/^/  - /'
    fi
    echo
  } >> "$OUT_DIR/summary.txt"
}

write_discovery_summary() {
  {
    echo "== discovered pages =="
    if [[ -s "$DISCOVERY_DIR/discovered-pages.txt" ]]; then
      sed 's/^/  - /' "$DISCOVERY_DIR/discovered-pages.txt"
    else
      echo "  - none"
    fi
    echo
    echo "== discovered scripts =="
    if [[ -s "$DISCOVERY_DIR/discovered-scripts.txt" ]]; then
      sed 's/^/  - /' "$DISCOVERY_DIR/discovered-scripts.txt"
    else
      echo "  - none"
    fi
    echo
    echo "== discovered help pages =="
    if [[ -s "$DISCOVERY_DIR/discovered-help-pages.txt" ]]; then
      sed 's/^/  - /' "$DISCOVERY_DIR/discovered-help-pages.txt"
    else
      echo "  - none"
    fi
    echo
    echo "== discovered cgis =="
    if [[ -s "$DISCOVERY_DIR/discovered-cgis.txt" ]]; then
      sed 's/^/  - /' "$DISCOVERY_DIR/discovered-cgis.txt"
    else
      echo "  - none"
    fi
    echo
  } >> "$OUT_DIR/summary.txt"
}

write_instructions() {
  cat > "$OUT_DIR/NEXT_STEPS.md" <<'EOF'
# Switching Evidence Capture

This script logs in to the switch, captures the menu and likely switching-related
pages, and saves any page, help page, script, or CGI names that appear in the
returned HTML and JavaScript.

Start with these files:

- `summary.txt`
- `discovery/discovered-pages.txt`
- `discovery/discovered-scripts.txt`
- `discovery/discovered-help-pages.txt`
- `discovery/discovered-cgis.txt`
- `bodies/Menu.htm`
- `bodies/str_menu.js`
- `bodies/menuList.js`
- `bodies/menu.js`
- `bodies/PortSettingRpm.htm`

What to look for:

1. Any page names containing `IGMP`, `Snoop`, `Lag`, `LAG`, or `Trunk`.
2. Any JavaScript files tied to those pages.
3. Any `*.cgi` handlers that appear in the page or script bodies.

If the GET capture reveals the right page names but not the write payloads,
finish with one real browser capture per feature:

1. Log in to the switch in a browser.
2. Open DevTools Network tab and preserve log.
3. Visit the identified page for:
   - Port Setting
   - IGMP Snooping
   - LAG / link aggregation
4. Make one minimal low-risk change.
5. Save or apply it.
6. Export the request details or a HAR file.

Most useful artifacts:

- full request URL
- method
- form payload
- response body
- updated page HTML after the change
EOF
}

require_env "TPLINK_EASYSMART_HOST"
require_env "TPLINK_EASYSMART_USERNAME"
require_env "TPLINK_EASYSMART_PASSWORD"

INSECURE_HTTP="${TPLINK_EASYSMART_INSECURE_HTTP:-true}"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="${TPLINK_EASYSMART_OUTPUT_DIR:-./switching-capture-$STAMP}"
COOKIE_JAR="$OUT_DIR/cookies.txt"
TRACE_DIR="$OUT_DIR/traces"
BODY_DIR="$OUT_DIR/bodies"
HEADER_DIR="$OUT_DIR/headers"
META_DIR="$OUT_DIR/meta"
DISCOVERY_DIR="$OUT_DIR/discovery"

mkdir -p "$TRACE_DIR" "$BODY_DIR" "$HEADER_DIR" "$META_DIR" "$DISCOVERY_DIR"

BASE_URL="$(build_base_url "$TPLINK_EASYSMART_HOST" "$INSECURE_HTTP")"

curl_common=(
  --silent
  --show-error
  --location
  --http1.1
  --cookie "$COOKIE_JAR"
  --cookie-jar "$COOKIE_JAR"
  --user-agent "Mozilla/5.0 (Macintosh; Intel Mac OS X) Switching Capture Script"
)

{
  echo "base_url=$BASE_URL"
  echo "host_env=$TPLINK_EASYSMART_HOST"
  echo "captured_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} > "$OUT_DIR/manifest.txt"

login

fetch_get "" "root.html"
fetch_get "Menu.htm" "Menu.htm"
fetch_get "str_menu.js" "str_menu.js"
fetch_get "menuList.js" "menuList.js"
fetch_get "menu.js" "menu.js"
fetch_get "PortSettingRpm.htm" "PortSettingRpm.htm"

seed_targets
discover_from_menu
fetch_candidate_pages
discover_from_pages
fetch_candidate_pages
fetch_discovered_scripts

: > "$OUT_DIR/summary.txt"
write_discovery_summary
record_summary "$BODY_DIR/logon.cgi.html" "logon.cgi.html" "$META_DIR/logon.cgi.html.status.txt"
record_summary "$BODY_DIR/root.html" "root.html" "$META_DIR/root.html.status.txt"
record_summary "$BODY_DIR/Menu.htm" "Menu.htm" "$META_DIR/Menu.htm.status.txt"
record_summary "$BODY_DIR/str_menu.js" "str_menu.js" "$META_DIR/str_menu.js.status.txt"
record_summary "$BODY_DIR/menuList.js" "menuList.js" "$META_DIR/menuList.js.status.txt"
record_summary "$BODY_DIR/menu.js" "menu.js" "$META_DIR/menu.js.status.txt"

while IFS= read -r page; do
  [[ -z "$page" ]] && continue
  if [[ -f "$BODY_DIR/$page" ]]; then
    record_summary "$BODY_DIR/$page" "$page" "$META_DIR/$page.status.txt"
  fi
done < "$DISCOVERY_DIR/candidate-pages.txt"

while IFS= read -r script_path; do
  [[ -z "$script_path" ]] && continue
  if [[ -f "$BODY_DIR/$script_path" ]]; then
    record_summary "$BODY_DIR/$script_path" "$script_path" "$META_DIR/$script_path.status.txt"
  fi
done < "$DISCOVERY_DIR/discovered-scripts.txt"

write_instructions

cat <<EOF
Capture complete.

Output directory: $OUT_DIR

Important files:
  $OUT_DIR/manifest.txt
  $OUT_DIR/summary.txt
  $OUT_DIR/NEXT_STEPS.md
  $DISCOVERY_DIR/discovered-pages.txt
  $DISCOVERY_DIR/discovered-scripts.txt
  $DISCOVERY_DIR/discovered-help-pages.txt
  $DISCOVERY_DIR/discovered-cgis.txt
  $BODY_DIR/Menu.htm
  $BODY_DIR/str_menu.js
  $BODY_DIR/menuList.js
  $BODY_DIR/menu.js
  $BODY_DIR/PortSettingRpm.htm
EOF
