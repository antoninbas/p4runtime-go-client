package client

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"

	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

func (c *Client) SetFwdPipeFromBytes(binBytes, p4infoBytes []byte) error {
	p4Info := &p4_config_v1.P4Info{}
	if err := proto.UnmarshalText(string(p4infoBytes), p4Info); err != nil {
		return fmt.Errorf("failed to decode P4Info Protobuf message: %v", err)
	}
	config := &p4_v1.ForwardingPipelineConfig{
		P4Info:         p4Info,
		P4DeviceConfig: binBytes,
	}
	req := &p4_v1.SetForwardingPipelineConfigRequest{
		DeviceId:   c.deviceID,
		ElectionId: &c.electionID,
		Action:     p4_v1.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT,
		Config:     config,
	}
	_, err := c.SetForwardingPipelineConfig(context.Background(), req)
	if err == nil {
		c.p4Info = p4Info
	}
	return err
}

func (c *Client) SetFwdPipe(binPath string, p4infoPath string) error {
	binBytes, err := ioutil.ReadFile(binPath)
	if err != nil {
		return fmt.Errorf("error when reading binary device config: %v", err)
	}
	p4infoBytes, err := ioutil.ReadFile(p4infoPath)
	if err != nil {
		return fmt.Errorf("error when reading P4Info text file: %v", err)
	}
	return c.SetFwdPipeFromBytes(binBytes, p4infoBytes)
}
