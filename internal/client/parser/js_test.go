package parser_test

import (
	"path/filepath"
	"testing"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/parser"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/testutil"
)

func TestExtractObjectSystemInfo(t *testing.T) {
	source := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "system_info.html"))

	object, err := parser.ExtractObject(source, "info_ds")
	if err != nil {
		t.Fatalf("ExtractObject() error = %v", err)
	}

	if got := object["firmwareStr"].([]any)[0].(string); got != "1.0.1 Build 20191204 Rel.71847" {
		t.Fatalf("unexpected firmware = %q", got)
	}
}

func TestExtractIntPortCount(t *testing.T) {
	source := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "port_setting.html"))

	value, err := parser.ExtractInt(source, "max_port_num")
	if err != nil {
		t.Fatalf("ExtractInt() error = %v", err)
	}
	if value != 8 {
		t.Fatalf("unexpected port count = %d", value)
	}
}

func TestExtractObjectVLANHexConversion(t *testing.T) {
	source := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "vlan_8021q.html"))

	object, err := parser.ExtractObject(source, "qvlan_ds")
	if err != nil {
		t.Fatalf("ExtractObject() error = %v", err)
	}

	tagMasks := object["tagMbrs"].([]any)
	if got := int(tagMasks[1].(float64)); got != 255 {
		t.Fatalf("unexpected tag mask = %d", got)
	}
}
