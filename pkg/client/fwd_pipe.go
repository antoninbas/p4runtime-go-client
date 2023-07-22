package client

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/prototext"

	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

type FwdPipeConfig struct {
	P4Info         *p4_config_v1.P4Info
	P4DeviceConfig []byte
	Cookie         uint64
}

func (c *Client) SetFwdPipeFromBytesWithAction(ctx context.Context, binBytes, p4infoBytes []byte, cookie uint64, action p4_v1.SetForwardingPipelineConfigRequest_Action) (*FwdPipeConfig, error) {
	p4Info := &p4_config_v1.P4Info{}
	if err := prototext.Unmarshal(p4infoBytes, p4Info); err != nil {
		return nil, fmt.Errorf("failed to decode P4Info Protobuf message: %v", err)
	}
	config := &p4_v1.ForwardingPipelineConfig{
		P4Info:         p4Info,
		P4DeviceConfig: binBytes,
		Cookie: &p4_v1.ForwardingPipelineConfig_Cookie{
			Cookie: cookie,
		},
	}

	req := &p4_v1.SetForwardingPipelineConfigRequest{
		DeviceId:   c.deviceID,
		ElectionId: c.electionID,
		Action:     action,
		Config:     config,
	}
	if c.role != nil {
		req.Role = c.role.Name
	}
	_, err := c.SetForwardingPipelineConfig(ctx, req)
	if err == nil {
		c.p4Info = p4Info
		return &FwdPipeConfig{
			P4Info:         p4Info,
			P4DeviceConfig: binBytes,
			Cookie:         cookie,
		}, nil
	}

	return nil, err
}

func (c *Client) SetFwdPipeFromBytes(ctx context.Context, binBytes, p4infoBytes []byte, cookie uint64) (*FwdPipeConfig, error) {
	return c.SetFwdPipeFromBytesWithAction(ctx, binBytes, p4infoBytes, cookie, p4_v1.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT)
}

func (c *Client) SaveFwdPipeFromBytes(ctx context.Context, binBytes, p4infoBytes []byte, cookie uint64) (*FwdPipeConfig, error) {
	return c.SetFwdPipeFromBytesWithAction(ctx, binBytes, p4infoBytes, cookie, p4_v1.SetForwardingPipelineConfigRequest_VERIFY_AND_SAVE)
}

func (c *Client) CommitFwdPipe(ctx context.Context) (*p4_v1.SetForwardingPipelineConfigResponse, error) {
	req := &p4_v1.SetForwardingPipelineConfigRequest{
		DeviceId:   c.deviceID,
		ElectionId: c.electionID,
		Action:     p4_v1.SetForwardingPipelineConfigRequest_COMMIT,
	}
	return c.SetForwardingPipelineConfig(ctx, req)
}

func (c *Client) SetFwdPipe(ctx context.Context, binPath string, p4infoPath string, cookie uint64) (*FwdPipeConfig, error) {
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		return nil, fmt.Errorf("error when reading binary device config: %v", err)
	}
	p4infoBytes, err := os.ReadFile(p4infoPath)
	if err != nil {
		return nil, fmt.Errorf("error when reading P4Info text file: %v", err)
	}
	return c.SetFwdPipeFromBytes(ctx, binBytes, p4infoBytes, cookie)
}

type GetFwdPipeResponseType int32

const (
	GetFwdPipeAll                   = GetFwdPipeResponseType(p4_v1.GetForwardingPipelineConfigRequest_ALL)
	GetFwdPipeCookieOnly            = GetFwdPipeResponseType(p4_v1.GetForwardingPipelineConfigRequest_COOKIE_ONLY)
	GetFwdPipeP4InfoAndCookie       = GetFwdPipeResponseType(p4_v1.GetForwardingPipelineConfigRequest_P4INFO_AND_COOKIE)
	GetFwdPipeDeviceConfigAndCookie = GetFwdPipeResponseType(p4_v1.GetForwardingPipelineConfigRequest_DEVICE_CONFIG_AND_COOKIE)
)

// GetFwdPipe retrieves the current pipeline config used in the remote switch.
//
// responseType is oneof:
//
//	GetFwdPipeAll, GetFwdPipeCookieOnly, GetFwdPipeP4InfoAndCookie, GetFwdPipeDeviceConfigAndCookie
//
// See https://p4.org/p4runtime/spec/v1.3.0/P4Runtime-Spec.html#sec-getforwardingpipelineconfig-rpc
func (c *Client) GetFwdPipe(ctx context.Context, responseType GetFwdPipeResponseType) (*FwdPipeConfig, error) {
	req := &p4_v1.GetForwardingPipelineConfigRequest{
		DeviceId:     c.deviceID,
		ResponseType: p4_v1.GetForwardingPipelineConfigRequest_ResponseType(responseType),
	}

	resp, err := c.GetForwardingPipelineConfig(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error when retrieving forwardingpipeline config: %v", err)
	}

	config := resp.GetConfig()
	if config == nil {
		// pipeline doesn't have a config yet
		return nil, nil
	}

	var pipeConfig = &FwdPipeConfig{
		P4Info:         config.GetP4Info(),
		P4DeviceConfig: config.GetP4DeviceConfig(),
	}
	if Cookie := config.GetCookie(); Cookie != nil {
		pipeConfig.Cookie = Cookie.GetCookie()
	}

	// save P4info for later use
	if pipeConfig.P4Info != nil {
		c.p4Info = pipeConfig.P4Info
	}

	return pipeConfig, nil
}
