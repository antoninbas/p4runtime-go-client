package client

import (
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	code "google.golang.org/genproto/googleapis/rpc/code"

	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

type Client struct {
	p4_v1.P4RuntimeClient
	deviceID   uint64
	electionID p4_v1.Uint128
	p4Info     *p4_config_v1.P4Info
}

func NewClient(p4RuntimeClient p4_v1.P4RuntimeClient, deviceID uint64, electionID p4_v1.Uint128) *Client {
	return &Client{
		P4RuntimeClient: p4RuntimeClient,
		deviceID:        deviceID,
		electionID:      electionID,
	}
}

func (c *Client) Run(stopCh <-chan struct{}, mastershipCh chan<- bool) error {
	stream, err := c.StreamChannel(context.Background())
	if err != nil {
		return fmt.Errorf("cannot establish stream: %v", err)
	}

	defer stream.CloseSend()

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// TODO: should reconnect
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive a stream message : %v", err)
			}
			arbitration, ok := in.Update.(*p4_v1.StreamMessageResponse_Arbitration)
			if !ok {
				// do not do anything with the message
				continue
			}
			if arbitration.Arbitration.Status.Code != int32(code.Code_OK) {
				if mastershipCh != nil {
					mastershipCh <- false
				}
			} else {
				if mastershipCh != nil {
					mastershipCh <- true
				}
			}
		}
	}()

	stream.Send(&p4_v1.StreamMessageRequest{
		Update: &p4_v1.StreamMessageRequest_Arbitration{&p4_v1.MasterArbitrationUpdate{
			DeviceId:   c.deviceID,
			ElectionId: &c.electionID,
		}},
	})

	<-stopCh
	return nil
}

func (c *Client) WriteUpdate(update *p4_v1.Update) error {
	req := &p4_v1.WriteRequest{
		DeviceId:   c.deviceID,
		ElectionId: &c.electionID,
		Updates:    []*p4_v1.Update{update},
	}
	_, err := c.Write(context.Background(), req)
	return err
}
