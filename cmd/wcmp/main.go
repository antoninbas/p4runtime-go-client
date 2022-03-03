package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"

	"github.com/antoninbas/p4runtime-go-client/pkg/client"
	"github.com/antoninbas/p4runtime-go-client/pkg/signals"
)

const (
	defaultDeviceID = 0
)

var (
	defaultAddr = fmt.Sprintf("127.0.0.1:%d", client.P4RuntimePort)
)

func nextHopToBytes(nhop string) []byte {
	b := []byte(nhop)
	padding := make([]byte, 32-len(b))
	return append(padding, b...)
}

func groupToBytes(group string) []byte {
	b := []byte(group)
	padding := make([]byte, 32-len(b))
	return append(padding, b...)
}

func insertOneGroup(ctx context.Context, p4RtC *client.Client, group string) error {
	mfs := []client.MatchInterface{&client.ExactMatch{
		Value: groupToBytes(group),
	}}
	watchPort := client.NewPortFromInt(1)
	actionSet := p4RtC.NewActionProfileActionSet()
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-56")}, 12, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-60")}, 12, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-57")}, 12, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-21")}, 12, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-61")}, 12, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-58")}, 12, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-59")}, 12, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-39")}, 12, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-63")}, 11, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-62")}, 10, watchPort)
	actionSet.AddAction("IngressImpl.set_nhop", [][]byte{nextHopToBytes("nexthop-64")}, 10, watchPort)
	entry := p4RtC.NewTableEntry("IngressImpl.wcmp_group", mfs, actionSet.TableAction(), nil)
	return p4RtC.InsertTableEntry(ctx, entry)
}

func deleteOneGroup(ctx context.Context, p4RtC *client.Client, group string) error {
	mfs := []client.MatchInterface{&client.ExactMatch{
		Value: groupToBytes(group),
	}}
	entry := p4RtC.NewTableEntry("IngressImpl.wcmp_group", mfs, nil, nil)
	return p4RtC.DeleteTableEntry(ctx, entry)
}

func main() {
	ctx := context.Background()

	var addr string
	flag.StringVar(&addr, "addr", defaultAddr, "P4Runtime server socket")
	var deviceID uint64
	flag.Uint64Var(&deviceID, "device-id", defaultDeviceID, "Device id")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode with debug log messages")
	var binPath string
	flag.StringVar(&binPath, "bin", "", "Path to P4 bin (not needed for bmv2 simple_switch_grpc)")
	var p4infoPath string
	flag.StringVar(&p4infoPath, "p4info", "", "Path to P4Info (not needed for bmv2 simple_switch_grpc)")

	flag.Parse()

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	binBytes := MustAsset("cmd/wcmp/wcmp.out/wcmp.json")
	p4infoBytes := MustAsset("cmd/wcmp/wcmp.out/p4info.pb.txt")

	if binPath != "" {
		var err error
		if binBytes, err = ioutil.ReadFile(binPath); err != nil {
			log.Fatalf("Error when reading binary config from '%s': %v", binPath, err)
		}
	}

	if p4infoPath != "" {
		var err error
		if p4infoBytes, err = ioutil.ReadFile(p4infoPath); err != nil {
			log.Fatalf("Error when reading P4Info text file '%s': %v", p4infoPath, err)
		}
	}

	log.Infof("Connecting to server at %s", addr)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Cannot connect to server: %v", err)
	}
	defer conn.Close()

	c := p4_v1.NewP4RuntimeClient(conn)
	resp, err := c.Capabilities(ctx, &p4_v1.CapabilitiesRequest{})
	if err != nil {
		log.Fatalf("Error in Capabilities RPC: %v", err)
	}
	log.Infof("P4Runtime server version is %s", resp.P4RuntimeApiVersion)

	stopCh := signals.RegisterSignalHandlers()

	electionID := p4_v1.Uint128{High: 0, Low: 1}

	p4RtC := client.NewClient(c, deviceID, electionID)
	arbitrationCh := make(chan bool)
	messageCh := make(chan *p4_v1.StreamMessageResponse, 1000)
	defer close(messageCh)
	go p4RtC.Run(stopCh, arbitrationCh, messageCh)

	waitCh := make(chan struct{})

	go func() {
		sent := false
		for isPrimary := range arbitrationCh {
			if isPrimary {
				log.Infof("We are the primary client!")
				if !sent {
					waitCh <- struct{}{}
					sent = true
				}
			} else {
				log.Infof("We are not the primary client!")
			}
		}
	}()

	func() {
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		select {
		case <-ctx.Done():
			log.Fatalf("Could not become the primary client within %v", timeout)
		case <-waitCh:
		}
	}()

	log.Info("Setting forwarding pipe")
	if _, err := p4RtC.SetFwdPipeFromBytes(ctx, binBytes, p4infoBytes, 0); err != nil {
		log.Fatalf("Error when setting forwarding pipe: %v", err)
	}

	log.Infof("Installing test groups")

	for i := 0; i < 100; i++ {
		group := fmt.Sprintf("group-%d", i)
		if err := insertOneGroup(ctx, p4RtC, group); err != nil {
			log.Errorf("Error when installing entry for '%s': %v", group, err)
		}
	}

	log.Infof("Deleting test groups")

	for i := 0; i < 100; i++ {
		group := fmt.Sprintf("group-%d", i)
		if err := deleteOneGroup(ctx, p4RtC, group); err != nil {
			log.Errorf("Error when removing entry for '%s': %v", group, err)
		}
	}

	log.Infof("Done")

	log.Info("Do Ctrl-C to quit")
	<-stopCh
	log.Info("Stopping client")
}
