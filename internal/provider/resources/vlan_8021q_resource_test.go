package resources

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/providerdata"
)

func TestVLANValidateConfigRejectsUnsupportedNames(t *testing.T) {
	vlanResource := &vlan8021qResource{}
	tests := []struct {
		name     string
		vlanName string
		wantErr  bool
	}{
		{name: "empty", vlanName: "", wantErr: true},
		{name: "ten ASCII bytes", vlanName: "1234567890"},
		{name: "eleven ASCII bytes", vlanName: "12345678901", wantErr: true},
		{name: "five multibyte characters", vlanName: strings.Repeat("é", 5)},
		{name: "six multibyte characters", vlanName: strings.Repeat("é", 6), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := resource.ValidateConfigResponse{}
			vlanResource.ValidateConfig(context.Background(), resource.ValidateConfigRequest{
				Config: vlanConfig(t, vlanResource, test.vlanName),
			}, &response)

			if got := response.Diagnostics.HasError(); got != test.wantErr {
				t.Fatalf("ValidateConfig(%q) error = %t, want %t: %v", test.vlanName, got, test.wantErr, response.Diagnostics)
			}
		})
	}
}

func TestVLANResourcesShareMutationLock(t *testing.T) {
	ctx := context.Background()
	providerData := &providerdata.Data{}
	first := &vlan8021qResource{}
	second := &vlan8021qResource{}

	for _, vlanResource := range []*vlan8021qResource{first, second} {
		response := resource.ConfigureResponse{}
		vlanResource.Configure(ctx, resource.ConfigureRequest{ProviderData: providerData}, &response)
		if response.Diagnostics.HasError() {
			t.Fatalf("Configure() diagnostics: %v", response.Diagnostics)
		}
	}

	if first.mutationMu == nil || second.mutationMu == nil || first.mutationMu != second.mutationMu {
		t.Fatal("configured VLAN resources must share one mutation lock")
	}

	first.lockMutation()
	acquired := make(chan struct{})
	go func() {
		second.lockMutation()
		close(acquired)
		second.unlockMutation()
	}()

	select {
	case <-acquired:
		t.Fatal("second VLAN resource acquired the lock while the first held it")
	case <-time.After(50 * time.Millisecond):
	}

	first.unlockMutation()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second VLAN resource did not acquire the lock after release")
	}
}

func vlanConfig(t *testing.T, vlanResource *vlan8021qResource, name string) tfsdk.Config {
	t.Helper()

	schemaResponse := resource.SchemaResponse{}
	vlanResource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)

	portSetType := tftypes.Set{ElementType: tftypes.Number}
	configType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id":             tftypes.String,
		"vlan_id":        tftypes.Number,
		"name":           tftypes.String,
		"tagged_ports":   portSetType,
		"untagged_ports": portSetType,
	}}

	return tfsdk.Config{
		Raw: tftypes.NewValue(configType, map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, nil),
			"vlan_id":        tftypes.NewValue(tftypes.Number, big.NewFloat(20)),
			"name":           tftypes.NewValue(tftypes.String, name),
			"tagged_ports":   tftypes.NewValue(portSetType, []tftypes.Value{}),
			"untagged_ports": tftypes.NewValue(portSetType, []tftypes.Value{}),
		}),
		Schema: schemaResponse.Schema,
	}
}
