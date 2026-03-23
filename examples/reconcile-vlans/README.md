# VLAN Reconciliation Example

This example is the real-switch workflow for taking over existing VLAN and PVID
state with Terraform before making changes.

It uses:

- `tplinkeasysmart_system_info` to prove connectivity
- `tplinkeasysmart_vlans` to inventory the current 802.1Q VLAN table
- `tplinkeasysmart_port_pvids` to inventory port PVID assignments
- `managed.tf.example` as a starting point for your switch-specific managed state
- `tplinkeasysmart_vlan_8021q` and `tplinkeasysmart_port_pvid` imports to
  reconcile live switch state into Terraform

## 1. Prepare secrets

```sh
cp secrets.sops.yaml.example secrets.sops.yaml
```

Edit `secrets.sops.yaml` with your switch credentials, then encrypt it with
SOPS. One simple age-based command is:

```sh
sops --encrypt --in-place --age "$(age-keygen -y ~/.config/sops/age/keys.txt)" secrets.sops.yaml
```

## 2. Build the local provider and initialize OpenTofu

The quickest path is to use the helper script in this directory:

```sh
./run.sh
```

It builds the provider into a repo-local mirror, resets the example
`.terraform.lock.hcl`, writes a temporary OpenTofu CLI config, runs `tofu init`,
and then runs an inventory-only `tofu apply` against the live data sources.

If you prefer to run the steps manually, use the commands below.

Reuse the local mirror pattern from the connection-check example:

```sh
ROOT_DIR="$(cd ../.. && pwd)"
EXAMPLE_DIR="$ROOT_DIR/examples/reconcile-vlans"
LOCAL_PROVIDER_VERSION="0.1.0"
TARGET_PLATFORM="$(go env GOOS)_$(go env GOARCH)"
MIRROR_ROOT="$EXAMPLE_DIR/terraform.d/plugins"
MIRROR_DIR="$MIRROR_ROOT/registry.terraform.io/lucavb/tplink-easysmart/$LOCAL_PROVIDER_VERSION/$TARGET_PLATFORM"
PROVIDER_BINARY="$MIRROR_DIR/terraform-provider-tplink-easysmart_v$LOCAL_PROVIDER_VERSION"
CLI_CONFIG_FILE="$ROOT_DIR/.opentofu/reconcile-vlans.tfrc"

mkdir -p "$MIRROR_DIR" "$(dirname "$CLI_CONFIG_FILE")"
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

TF_CLI_CONFIG_FILE="$CLI_CONFIG_FILE" tofu -chdir="$EXAMPLE_DIR" init
```

## 3. Inventory the live switch

```sh
TF_CLI_CONFIG_FILE="$CLI_CONFIG_FILE" tofu -chdir="$EXAMPLE_DIR" apply
```

Review the outputs:

- `switch_identity`
- `vlan_inventory`
- `port_pvid_inventory`

These outputs are the source of truth for the initial Terraform config.

## 4. Managed resources

Copy `managed.tf.example` to `managed.tf`, then replace the sample VLANs and
PVIDs with the inventory captured from your own switch.

Tips:

- Manage every live VLAN you want Terraform to own.
- For each imported `tplinkeasysmart_port_pvid`, ensure the referenced VLAN
  exists in your `tplinkeasysmart_vlan_8021q` configuration.
- Keep your management/uplink path out of the first write test.

## 5. Import the existing switch state

Copy `import-current-state.sh.example` to `import-current-state.sh`, customize
the import commands for your switch, then import the live switch objects into
the resources in `managed.tf`:

```sh
cp import-current-state.sh.example import-current-state.sh
chmod +x import-current-state.sh
./import-current-state.sh
```

Import IDs are:

- VLAN resource: the numeric `vlan_id`
- Port PVID resource: the numeric `port_id`

Your customized script should import all current VLAN and PVID resources and
then run:

```sh
TF_CLI_CONFIG_FILE="$CLI_CONFIG_FILE" tofu -chdir="$EXAMPLE_DIR" plan
```

The goal is a clean no-op plan. If OpenTofu shows changes, compare them against
the inventory outputs and correct `managed.tf` until the plan is empty.

## 6. Run one controlled VLAN write test

After you have a no-op plan:

1. Pick one unused VLAN ID.
2. Pick one safe untagged port and one safe tagged port.
3. Add or edit a single `tplinkeasysmart_vlan_8021q` resource.
4. Optionally add one `tplinkeasysmart_port_pvid` change for the untagged test
   port.
5. Run `tofu plan`, `tofu apply`, then `tofu plan` again.

The final `plan` should be empty.

## Safety notes

- Ensure 802.1Q VLAN mode is enabled before write tests.
- Avoid deleting VLANs in the first round of live testing.
- Avoid ports that carry management traffic until you have already proven
  no-op reconciliation works.
