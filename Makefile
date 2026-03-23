test:
	go test ./...

testacc:
	TF_ACC=1 go test ./... -count=1

lint:
	go vet ./...

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name tplinkeasysmart --rendered-provider-name terraform-provider-tplink-easysmart

.PHONY: test testacc lint docs
