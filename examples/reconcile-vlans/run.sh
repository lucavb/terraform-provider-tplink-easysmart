#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EXAMPLE_DIR="$ROOT_DIR/examples/reconcile-vlans"
SECRETS_FILE="$EXAMPLE_DIR/secrets.sops.yaml"
LOCAL_PROVIDER_VERSION="0.1.0"
CLI_CONFIG_DIR="$ROOT_DIR/.opentofu"
CLI_CONFIG_FILE="$CLI_CONFIG_DIR/reconcile-vlans.tfrc"

for cmd in go tofu sops; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "missing required command: $cmd" >&2
    exit 1
  fi
done

TARGET_OS="$(go env GOOS)"
TARGET_ARCH="$(go env GOARCH)"
TARGET_PLATFORM="${TARGET_OS}_${TARGET_ARCH}"
MIRROR_ROOT="$EXAMPLE_DIR/terraform.d/plugins"
MIRROR_DIR="$MIRROR_ROOT/registry.terraform.io/lucavb/tplink-easysmart/$LOCAL_PROVIDER_VERSION/$TARGET_PLATFORM"
PROVIDER_BINARY="$MIRROR_DIR/terraform-provider-tplink-easysmart_v$LOCAL_PROVIDER_VERSION"
LOCK_FILE="$EXAMPLE_DIR/.terraform.lock.hcl"

if [[ ! -f "$SECRETS_FILE" ]]; then
  cat >&2 <<EOF
missing encrypted secrets file: $SECRETS_FILE

Create it from the example first:
  cp "$EXAMPLE_DIR/secrets.sops.yaml.example" "$SECRETS_FILE"

Then edit and encrypt it, for example with age:
  sops --encrypt --in-place --age "\$(age-keygen -y ~/.config/sops/age/keys.txt)" "$SECRETS_FILE"
EOF
  exit 1
fi

rm -rf "$EXAMPLE_DIR/terraform.d/plugins/registry.terraform.io/lucavb/tplink-easysmart"
rm -f "$LOCK_FILE"
mkdir -p "$MIRROR_DIR"
mkdir -p "$CLI_CONFIG_DIR"

echo "Building local provider binary for $TARGET_PLATFORM..."
go build -o "$PROVIDER_BINARY" "$ROOT_DIR"
chmod +x "$PROVIDER_BINARY"

cat >"$CLI_CONFIG_FILE" <<EOF
provider_installation {
  filesystem_mirror {
    path    = "$MIRROR_ROOT"
    include = ["registry.terraform.io/lucavb/tplink-easysmart"]
  }

  direct {
    exclude = ["registry.terraform.io/lucavb/tplink-easysmart"]
  }
}
EOF

echo "Using repo-local provider mirror: $MIRROR_DIR"
echo "Using temporary OpenTofu CLI config: $CLI_CONFIG_FILE"
echo "Reset example lockfile: $LOCK_FILE"
echo "Initializing reconcile-vlans example with OpenTofu..."
echo "Running inventory-only apply for live data sources..."
(
  cd "$EXAMPLE_DIR"
  TF_CLI_CONFIG_FILE="$CLI_CONFIG_FILE" tofu init
  TF_CLI_CONFIG_FILE="$CLI_CONFIG_FILE" tofu apply \
    -target=data.tplinkeasysmart_system_info.this \
    -target=data.tplinkeasysmart_vlans.this \
    -target=data.tplinkeasysmart_port_pvids.this \
    "$@"
)
