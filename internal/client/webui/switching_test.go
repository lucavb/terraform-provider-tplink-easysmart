package webui_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/webui"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/testutil"
)

func TestReadSwitchingState(t *testing.T) {
	loginSuccess := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_success.html"))
	igmpPage := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "igmp_snooping.html"))
	trunkPage := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "port_trunk.html"))

	igmpPage = strings.Replace(igmpPage, "count:0,", "count:1,", 1)
	igmpPage = strings.Replace(igmpPage, "ipStr:[\n\n],", "ipStr:[\n3232235777\n],", 1)
	igmpPage = strings.Replace(igmpPage, "vlanStr:[\n\n],", "vlanStr:[\n20\n],", 1)
	igmpPage = strings.Replace(igmpPage, "portStr:[\n\n],", "portStr:[\n19\n],", 1)
	igmpPage = strings.Replace(igmpPage, "0x0,0x0", "0x3,0x0", 1)

	trunkPage = strings.Replace(trunkPage, "portStr_g1:[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]", "portStr_g1:[1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]", 1)
	trunkPage = strings.Replace(trunkPage, "portStr_g2:[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]", "portStr_g2:[0,0,0,0,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]", 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/logon.cgi":
			_, _ = w.Write([]byte(loginSuccess))
		case "/IgmpSnoopingRpm.htm":
			_, _ = w.Write([]byte(igmpPage))
		case "/PortTrunkRpm.htm":
			_, _ = w.Write([]byte(trunkPage))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	switchClient := webui.New(client.Config{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "password",
		Timeout:  2 * time.Second,
	})

	ctx := context.Background()
	if err := switchClient.Authenticate(ctx); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	igmp, err := switchClient.GetIGMPSnooping(ctx)
	if err != nil {
		t.Fatalf("GetIGMPSnooping() error = %v", err)
	}
	if !igmp.Enabled || igmp.ReportMessageSuppression {
		t.Fatalf("unexpected igmp state = %+v", igmp)
	}
	if len(igmp.Groups) != 1 {
		t.Fatalf("unexpected igmp groups = %#v", igmp.Groups)
	}
	if igmp.Groups[0].IPAddress != "192.168.1.1" || igmp.Groups[0].VLANID != 20 {
		t.Fatalf("unexpected igmp group = %+v", igmp.Groups[0])
	}
	if len(igmp.Groups[0].Ports) != 1 || igmp.Groups[0].Ports[0] != 5 {
		t.Fatalf("unexpected igmp ports = %#v", igmp.Groups[0].Ports)
	}
	if len(igmp.Groups[0].LAGGroups) != 1 || igmp.Groups[0].LAGGroups[0] != 1 {
		t.Fatalf("unexpected igmp lag groups = %#v", igmp.Groups[0].LAGGroups)
	}

	lags, err := switchClient.GetLAGs(ctx)
	if err != nil {
		t.Fatalf("GetLAGs() error = %v", err)
	}
	if lags.MaxGroups != 2 || lags.PortCount != 8 || lags.PortsPerGroup != 4 {
		t.Fatalf("unexpected lag info = %+v", lags)
	}
	if len(lags.Groups) != 2 {
		t.Fatalf("unexpected lag groups = %#v", lags.Groups)
	}
	if got := lags.Groups[0].Ports; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("unexpected lag group 1 ports = %#v", got)
	}
	if got := lags.Groups[1].Ports; len(got) != 2 || got[0] != 5 || got[1] != 6 {
		t.Fatalf("unexpected lag group 2 ports = %#v", got)
	}
}

func TestWriteSwitchingQueries(t *testing.T) {
	loginSuccess := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_success.html"))
	igmpPage := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "igmp_snooping.html"))
	trunkPage := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "port_trunk.html"))

	var gotPaths []string
	var requests []capturedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotPaths = append(gotPaths, r.URL.RequestURI())
		requests = append(requests, capturedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
		})

		switch r.URL.Path {
		case "/logon.cgi":
			_, _ = w.Write([]byte(loginSuccess))
		case "/IgmpSnoopingRpm.htm", "/igmpSnooping.cgi":
			_, _ = w.Write([]byte(igmpPage))
		case "/PortTrunkRpm.htm", "/port_trunk_set.cgi", "/port_trunk_display.cgi":
			_, _ = w.Write([]byte(trunkPage))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	switchClient := webui.New(client.Config{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "password",
		Timeout:  2 * time.Second,
	})

	ctx := context.Background()
	if err := switchClient.Authenticate(ctx); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if err := switchClient.UpdateIGMPSnooping(ctx, true, true); err != nil {
		t.Fatalf("UpdateIGMPSnooping() error = %v", err)
	}
	if err := switchClient.UpsertLAG(ctx, 1, []int{2, 1}); err != nil {
		t.Fatalf("UpsertLAG() error = %v", err)
	}
	if err := switchClient.DeleteLAG(ctx, 2); err != nil {
		t.Fatalf("DeleteLAG() error = %v", err)
	}

	assertPathSeen(t, gotPaths, "/igmpSnooping.cgi?Apply=Apply&igmp_mode=1&reportSu_mode=1")
	assertPathSeen(t, gotPaths, "/port_trunk_set.cgi?groupId=1&portid=1&portid=2&setapply=Apply")
	assertPathSeen(t, gotPaths, "/port_trunk_display.cgi?chk_trunk=2&setDelete=Delete")

	for _, req := range requests {
		if req.Path == "/igmpSnooping.cgi" && req.Method != http.MethodGet {
			t.Fatalf("expected GET for igmpSnooping.cgi, got %s", req.Method)
		}
	}
}
