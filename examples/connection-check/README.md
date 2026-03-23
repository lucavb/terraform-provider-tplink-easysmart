# Connection Check Example

This example uses the `sops` OpenTofu provider to read the switch host,
username, and password from a local SOPS-encrypted YAML file, then reads
`tplinkeasysmart_system_info` to prove the provider can connect.

The repo-local `scripts/connection-check.sh` script builds the provider into
`terraform.d/plugins` under this example and writes a temporary OpenTofu CLI
config that installs `registry.terraform.io/lucavb/tplink-easysmart` from that
repo-local filesystem mirror without touching your global OpenTofu CLI config.

## Files

- `main.tf`: minimal provider and data source example
- `secrets.sops.yaml.example`: template for your local secret file
- `secrets.sops.yaml`: your real encrypted secret file, ignored by git

## 1. Create the local secret file

```sh
cp secrets.sops.yaml.example secrets.sops.yaml
```

Edit `secrets.sops.yaml` with your switch values, then encrypt it with SOPS.
One simple age-based command is:

```sh
sops --encrypt --in-place --age "$(age-keygen -y ~/.config/sops/age/keys.txt)" secrets.sops.yaml
```

## 2. Run OpenTofu

```sh
../../scripts/connection-check.sh
```

The script rebuilds the local provider for your current platform before running
`tofu init` and `tofu apply`.

If the connection works, Terraform will print a `switch_identity` output with
basic system information from the switch.
