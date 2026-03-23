package client

import (
	"context"
	"net/http"
	"time"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/model"
)

type Config struct {
	BaseURL    string
	Username   string
	Password   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Client interface {
	Authenticate(context.Context) error
	GetSystemInfo(context.Context) (model.SystemInfo, error)
	GetPorts(context.Context) ([]model.Port, error)
	GetManagementIP(context.Context) (model.ManagementIP, error)
	GetVLANs(context.Context) (model.VLANTable, error)
	GetPVIDs(context.Context) ([]model.PortPVID, error)
	UpsertVLAN(context.Context, int, string, []int, []int) error
	DeleteVLAN(context.Context, int) error
	SetPortPVID(context.Context, int, int) error
	UpdatePortSettings(context.Context, int, *bool, *int, *int) error
}
