package p4switch

import (
	"context"
	"github.com/antoninbas/p4runtime-go-client/pkg/client"
	"fmt"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultPort = 50050
	defaultAddr = "127.0.0.1"
	defaultWait = 250 * time.Millisecond
)

func CreateSwitch(deviceID uint64, configName string, ports int, certFile string) *GrpcSwitch {
	return &GrpcSwitch{
		id:                deviceID,
		initialConfigName: configName,
		ports:             ports,
		addr:              fmt.Sprintf("%s:%d", defaultAddr, defaultPort+deviceID),
		log:               log.WithField("ID", deviceID),
		certFile:          certFile,
	}
	// GrpcSwitch (sw.go) contiene anche:
	//	- p4RtC      *client.Client		--inizializzato in RunSwitch
	//	- messageCh  chan *p4_v1.StreamMessageResponse
}

func (sw *GrpcSwitch) RunSwitch(ct context.Context) error {

	sw.log.Infof("Connecting to server at %s", sw.addr)
	var creds credentials.TransportCredentials
	if sw.certFile != "" {
		var err error
		if creds, err = credentials.NewClientTLSFromFile(sw.certFile, ""); err != nil {
			return err
		}
	} else {
		creds = insecure.NewCredentials()
	}
	conn, err := grpc.Dial(sw.addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return err
	}

	// create context
	ctx, cancel := context.WithCancel(ct)

	// checking runtime
	c := p4_v1.NewP4RuntimeClient(conn)
	resp, err := c.Capabilities(ctx, &p4_v1.CapabilitiesRequest{})
	if err != nil {
		cancel()
		return err
	}
	sw.log.Debugf("Connected, runtime version: %s", resp.P4RuntimeApiVersion)

	// create runtime client
	electionID := p4_v1.Uint128{High: 0, Low: 1}
	sw.messageCh = make(chan *p4_v1.StreamMessageResponse, 1000)
	arbitrationCh := make(chan bool)
	sw.p4RtC = client.NewClient(c, sw.id, &electionID) //Inizializzo sw.p4RtC a nuovo oggetto CLIENT
	go sw.p4RtC.Run(ctx, conn, arbitrationCh, sw.messageCh)
	// check primary
	for isPrimary := range arbitrationCh {
		if isPrimary {
			log.Trace("we are the primary client")
			break
		} else {
			cancel()
			return fmt.Errorf("we are not the primary client")
		}
	}
	// set pipeline config
	time.Sleep(defaultWait)
	if _, err := sw.p4RtC.SetFwdPipeFromBytes(ctx,sw.readBin(), sw.readP4Info(), 0); err != nil {
		cancel()
		return err
	}
	sw.log.Debug("Setted forwarding pipe")
	//
	sw.errCh = make(chan error, 1)
	go sw.handleStreamMessages(ctx)
	go sw.startRunner(ctx, cancel)
	//
	sw.InitiateConfig(ctx)
	sw.EnableDigest(ctx)
	//
	sw.log.Info("Switch started")
	return nil
}

func (sw *GrpcSwitch) startRunner(ctx context.Context, cancel context.CancelFunc) {
	defer func() {
		close(sw.messageCh)
		cancel()
		sw.log.Info("Stopping")
	}()
	for {
		select {
		case err := <-sw.errCh:
			sw.log.Errorf("%v", err)
			cancel()
		case <-ctx.Done():
			return
		}
	}
}

func (sw *GrpcSwitch) handleStreamMessages(ctx context.Context) {
	for message := range sw.messageCh {
		switch m := message.Update.(type) {
		case *p4_v1.StreamMessageResponse_Packet:
			sw.log.Debug("Received Packetin")
		case *p4_v1.StreamMessageResponse_Digest:
			sw.log.Trace("Received DigestList")
			sw.HandleDigest(ctx,m.Digest)
		case *p4_v1.StreamMessageResponse_IdleTimeoutNotification:
			sw.log.Trace("Received IdleTimeoutNotification")
		case *p4_v1.StreamMessageResponse_Error:
			sw.log.Trace("Received StreamError")
			sw.errCh <- fmt.Errorf("StreamError: %v", m.Error)
		default:
			sw.log.Debug("Received unknown stream message")
		}
	}
	sw.log.Trace("Closed message channel")
}
