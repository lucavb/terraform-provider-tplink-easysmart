package webui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/webui"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/testutil"
)

func TestAuthenticateAndReadSystemInfo(t *testing.T) {
	loginSuccess := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_success.html"))
	systemInfo := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "system_info.html"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/logon.cgi":
			_, _ = w.Write([]byte(loginSuccess))
		case "/SystemInfoRpm.htm":
			_, _ = w.Write([]byte(systemInfo))
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

	info, err := switchClient.GetSystemInfo(ctx)
	if err != nil {
		t.Fatalf("GetSystemInfo() error = %v", err)
	}

	if info.Description != "TL-SG108PE" {
		t.Fatalf("unexpected model = %q", info.Description)
	}
}

func TestAuthenticateFailure(t *testing.T) {
	loginFailure := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_failure.html"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(loginFailure))
	}))
	defer server.Close()

	switchClient := webui.New(client.Config{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "wrong",
		Timeout:  2 * time.Second,
	})

	if err := switchClient.Authenticate(context.Background()); err == nil {
		t.Fatal("Authenticate() expected error, got nil")
	}
}

func TestReadManagementIPAndVLANs(t *testing.T) {
	loginSuccess := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_success.html"))
	ipSetting := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "ip_setting.html"))
	vlanSetting := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "vlan_8021q.html"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/logon.cgi":
			_, _ = w.Write([]byte(loginSuccess))
		case "/IpSettingRpm.htm":
			_, _ = w.Write([]byte(ipSetting))
		case "/Vlan8021QRpm.htm":
			_, _ = w.Write([]byte(vlanSetting))
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

	ipInfo, err := switchClient.GetManagementIP(ctx)
	if err != nil {
		t.Fatalf("GetManagementIP() error = %v", err)
	}
	if ipInfo.IP != "10.0.2.1" || ipInfo.VLAN != 1 {
		t.Fatalf("unexpected management IP state = %+v", ipInfo)
	}

	vlans, err := switchClient.GetVLANs(ctx)
	if err != nil {
		t.Fatalf("GetVLANs() error = %v", err)
	}
	if len(vlans.VLANs) != 6 {
		t.Fatalf("unexpected VLAN count = %d", len(vlans.VLANs))
	}
	if got := vlans.VLANs[5].TaggedPorts; len(got) == 0 || got[0] != 1 {
		t.Fatalf("unexpected tagged ports = %#v", got)
	}
}

func TestReadPVIDs(t *testing.T) {
	loginSuccess := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_success.html"))
	pvidPage := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "vlan_pvid.html"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/logon.cgi":
			_, _ = w.Write([]byte(loginSuccess))
		case "/Vlan8021QPvidRpm.htm":
			_, _ = w.Write([]byte(pvidPage))
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

	pvids, err := switchClient.GetPVIDs(ctx)
	if err != nil {
		t.Fatalf("GetPVIDs() error = %v", err)
	}
	if len(pvids) != 8 || pvids[1].PVID != 1 {
		t.Fatalf("unexpected PVIDs = %#v", pvids)
	}
}

func TestWriteQueries(t *testing.T) {
	loginSuccess := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_success.html"))
	vlanSetting := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "vlan_8021q.html"))
	pvidPage := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "vlan_pvid.html"))
	portSetting := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "port_setting.html"))

	var gotPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.RequestURI())
		switch r.URL.Path {
		case "/logon.cgi":
			_, _ = w.Write([]byte(loginSuccess))
		case "/Vlan8021QRpm.htm", "/qvlanSet.cgi":
			_, _ = w.Write([]byte(vlanSetting))
		case "/Vlan8021QPvidRpm.htm", "/vlanPvidSet.cgi":
			_, _ = w.Write([]byte(pvidPage))
		case "/PortSettingRpm.htm", "/port_setting.cgi":
			_, _ = w.Write([]byte(portSetting))
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

	if err := switchClient.UpsertVLAN(ctx, 4093, "TfLab", []int{3}, []int{2}); err != nil {
		t.Fatalf("UpsertVLAN() error = %v", err)
	}
	if err := switchClient.DeleteVLAN(ctx, 4093); err != nil {
		t.Fatalf("DeleteVLAN() error = %v", err)
	}
	if err := switchClient.SetPortPVID(ctx, 2, 5); err != nil {
		t.Fatalf("SetPortPVID() error = %v", err)
	}
	flow := 1
	if err := switchClient.UpdatePortSettings(ctx, 2, nil, nil, &flow); err != nil {
		t.Fatalf("UpdatePortSettings() error = %v", err)
	}

	assertPathSeen(t, gotPaths, "/qvlanSet.cgi?qvlan_add=Add%2FModify&selType_1=2&selType_2=0&selType_3=1&selType_4=2&selType_5=2&selType_6=2&selType_7=2&selType_8=2&vid=4093&vname=TfLab")
	assertPathSeen(t, gotPaths, "/qvlanSet.cgi?qvlan_del=Delete&selVlans=4093")
	assertPathSeen(t, gotPaths, "/vlanPvidSet.cgi?pbm=2&pvid=5")
	assertPathSeen(t, gotPaths, "/port_setting.cgi?apply=Apply&flowcontrol=1&portid=2&speed=7&state=7")
}

func assertPathSeen(t *testing.T, gotPaths []string, want string) {
	t.Helper()

	for _, got := range gotPaths {
		if got == want {
			return
		}
	}

	t.Fatalf("request %q not seen in %#v", want, gotPaths)
}
