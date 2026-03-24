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

type capturedRequest struct {
	Method string
	Path   string
	Body   string
}

func TestReadQoSState(t *testing.T) {
	loginSuccess := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_success.html"))
	qosBasic := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "qos_basic.html"))
	qosBandwidth := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "qos_bandwidth.html"))
	qosStorm := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "qos_storm.html"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/logon.cgi":
			_, _ = w.Write([]byte(loginSuccess))
		case "/QosBasicRpm.htm":
			_, _ = w.Write([]byte(qosBasic))
		case "/QosBandWidthControlRpm.htm":
			_, _ = w.Write([]byte(qosBandwidth))
		case "/QosStormControlRpm.htm":
			_, _ = w.Write([]byte(qosStorm))
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

	mode, err := switchClient.GetQoSMode(ctx)
	if err != nil {
		t.Fatalf("GetQoSMode() error = %v", err)
	}
	if mode.ID != "qos" || mode.Mode != 2 {
		t.Fatalf("unexpected qos mode = %+v", mode)
	}

	priorities, err := switchClient.GetPortQoSPriorities(ctx)
	if err != nil {
		t.Fatalf("GetPortQoSPriorities() error = %v", err)
	}
	if len(priorities) != 8 || priorities[0].Priority != 1 {
		t.Fatalf("unexpected priorities = %#v", priorities)
	}

	bandwidthControls, err := switchClient.GetPortBandwidthControls(ctx)
	if err != nil {
		t.Fatalf("GetPortBandwidthControls() error = %v", err)
	}
	if len(bandwidthControls) != 8 || bandwidthControls[0].IngressRateKbps != 0 || bandwidthControls[0].EgressRateKbps != 0 {
		t.Fatalf("unexpected bandwidth controls = %#v", bandwidthControls)
	}

	stormControls, err := switchClient.GetPortStormControls(ctx)
	if err != nil {
		t.Fatalf("GetPortStormControls() error = %v", err)
	}
	if len(stormControls) != 8 || stormControls[0].Enabled || len(stormControls[0].StormTypes) != 0 {
		t.Fatalf("unexpected storm controls = %#v", stormControls)
	}
}

func TestWriteQoSForms(t *testing.T) {
	loginSuccess := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "login_success.html"))
	qosBasic := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "qos_basic.html"))
	qosBandwidth := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "qos_bandwidth.html"))
	qosStorm := testutil.ReadFixture(t, filepath.Join("internal", "testing", "fixtures", "qos_storm.html"))

	qosBasicPortBased := strings.Replace(qosBasic, "var qosMode = 2;", "var qosMode = 0;", 1)
	qosBasicPortBased = strings.Replace(qosBasicPortBased, "var pTrunk = new Array(0,0,0,0,0,0,0,0);", "var pTrunk = new Array(1,1,0,0,0,0,0,0);", 1)
	qosBandwidthLag := strings.Replace(qosBandwidth, "var bcInfo = new Array(\n0,0,0,\n0,0,0,\n", "var bcInfo = new Array(\n0,0,1,\n0,0,1,\n", 1)
	qosStormLag := strings.Replace(qosStorm, "var scInfo = new Array(\n0,0,0,\n0,0,0,\n", "var scInfo = new Array(\n0,0,1,\n0,0,1,\n", 1)

	var requests []capturedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, capturedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
		})

		switch r.URL.Path {
		case "/logon.cgi":
			_, _ = w.Write([]byte(loginSuccess))
		case "/QosBasicRpm.htm", "/qos_mode_set.cgi", "/qos_port_priority_set.cgi":
			_, _ = w.Write([]byte(qosBasicPortBased))
		case "/QosBandWidthControlRpm.htm", "/qos_bandwidth_set.cgi":
			_, _ = w.Write([]byte(qosBandwidthLag))
		case "/QosStormControlRpm.htm", "/qos_storm_set.cgi":
			_, _ = w.Write([]byte(qosStormLag))
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

	if err := switchClient.UpdateQoSMode(ctx, 1); err != nil {
		t.Fatalf("UpdateQoSMode() error = %v", err)
	}
	if err := switchClient.SetPortQoSPriority(ctx, 1, 4); err != nil {
		t.Fatalf("SetPortQoSPriority() error = %v", err)
	}
	if err := switchClient.SetPortBandwidthControl(ctx, 1, 1234, 5678); err != nil {
		t.Fatalf("SetPortBandwidthControl() error = %v", err)
	}
	if err := switchClient.SetPortStormControl(ctx, 1, true, 900, []int{4, 1}); err != nil {
		t.Fatalf("SetPortStormControl(enabled) error = %v", err)
	}
	if err := switchClient.SetPortStormControl(ctx, 3, false, 0, nil); err != nil {
		t.Fatalf("SetPortStormControl(disabled) error = %v", err)
	}

	assertQoSRequestSeen(t, requests, "POST", "/qos_mode_set.cgi", "qosmode=Apply&rd_qosmode=1")
	assertQoSRequestSeen(t, requests, "POST", "/qos_port_priority_set.cgi", "apply=Apply&port_queue=3&sel_1=1&sel_2=1")
	assertQoSRequestSeen(t, requests, "POST", "/qos_bandwidth_set.cgi", "applay=Apply&egrRate=5678&igrRate=1234&sel_1=1&sel_2=1")
	assertQoSRequestSeen(t, requests, "POST", "/qos_storm_set.cgi", "applay=Apply&rate=900&sel_1=1&sel_2=1&state=1&stormType=1&stormType=4")
	assertQoSRequestSeen(t, requests, "POST", "/qos_storm_set.cgi", "applay=Apply&sel_3=1&state=0")
}

func assertQoSRequestSeen(t *testing.T, requests []capturedRequest, wantMethod string, wantPath string, wantBody string) {
	t.Helper()

	for _, req := range requests {
		if req.Method == wantMethod && req.Path == wantPath && req.Body == wantBody {
			return
		}
	}

	t.Fatalf("request %s %s with body %q not seen in %#v", wantMethod, wantPath, wantBody, requests)
}
