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

const (
	P4RuntimePort = 9559
)

type ClientOptions struct {
	CanonicalBytestrings bool
}

var defaultClientOptions = ClientOptions{
	CanonicalBytestrings: true,
}

func DisableCanonicalBytestrings(options *ClientOptions) {
	options.CanonicalBytestrings = false
}

type Client struct {
	ClientOptions
	p4_v1.P4RuntimeClient
	deviceID     uint64
	electionID   p4_v1.Uint128
	p4Info       *p4_config_v1.P4Info
	streamSendCh chan *p4_v1.StreamMessageRequest
}

func NewClient(
	p4RuntimeClient p4_v1.P4RuntimeClient,
	deviceID uint64,
	electionID p4_v1.Uint128,
	optionsModifierFns ...func(*ClientOptions),
) *Client {
	options := defaultClientOptions
	for _, fn := range optionsModifierFns {
		fn(&options)
	}
	return &Client{
		ClientOptions:   options,
		P4RuntimeClient: p4RuntimeClient,
		deviceID:        deviceID,
		electionID:      electionID,
		streamSendCh:    make(chan *p4_v1.StreamMessageRequest, 1000), // TODO: should be configurable
	}
}

func (c *Client) Run(
	stopCh <-chan struct{},
	arbitrationCh chan<- bool,
	messageCh chan<- *p4_v1.StreamMessageResponse, // all other stream messages besides arbitration
) error {
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
				messageCh <- in
				continue
			}
			if arbitration.Arbitration.Status.Code != int32(code.Code_OK) {
				if arbitrationCh != nil {
					arbitrationCh <- false
				}
			} else {
				if arbitrationCh != nil {
					arbitrationCh <- true
				}
			}
		}
	}()

	stream.Send(&p4_v1.StreamMessageRequest{
		Update: &p4_v1.StreamMessageRequest_Arbitration{Arbitration: &p4_v1.MasterArbitrationUpdate{
			DeviceId:   c.deviceID,
			ElectionId: &c.electionID,
		}},
	})

	for {
		select {
		case m := <-c.streamSendCh:
			stream.Send(m)
		case <-stopCh:
			break
		}
	}

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

func (c *Client) ReadEntitySingle(entity *p4_v1.Entity) (*p4_v1.Entity, error) {
	req := &p4_v1.ReadRequest{
		DeviceId: c.deviceID,
		Entities: []*p4_v1.Entity{entity},
	}
	stream, err := c.Read(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	var readEntity *p4_v1.Entity
	count := 0
	for {
		rep, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		for _, e := range rep.Entities {
			count++
			readEntity = e
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("expected a single entity but got none")
	}
	if count > 1 {
		return nil, fmt.Errorf("expected a single entity but got several")
	}
	return readEntity, nil
}

// ReadEntityWildcard will block and send all read entities on readEntityCh. It will close the
// channel when the RPC completes and return any error that may have occurred.
func (c *Client) ReadEntityWildcard(entity *p4_v1.Entity, readEntityCh chan<- *p4_v1.Entity) error {
	defer close(readEntityCh)

	req := &p4_v1.ReadRequest{
		DeviceId: c.deviceID,
		Entities: []*p4_v1.Entity{entity},
	}
	stream, err := c.Read(context.TODO(), req)
	if err != nil {
		return err
	}
	for {
		rep, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		for _, e := range rep.Entities {
			readEntityCh <- e
		}
	}
	return nil
}
