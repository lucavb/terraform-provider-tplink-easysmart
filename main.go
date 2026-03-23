package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	tplinkeasysmart "github.com/lucavb/terraform-provider-tplink-easysmart/internal/provider"
)

var version = "dev"

func main() {
	err := providerserver.Serve(context.Background(), tplinkeasysmart.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/lucavb/tplink-easysmart",
	})
	if err != nil {
		log.Fatal(err)
	}
}
