package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"

	"github.com/antoninbas/p4runtime-go-client/pkg/client"
	"github.com/antoninbas/p4runtime-go-client/pkg/signals"
	"github.com/antoninbas/p4runtime-go-client/pkg/util/conversion"
)

const (
	defaultAddr     = "127.0.0.1:50051"
	defaultDeviceID = 0
	mgrp            = 0xab
	macTimeout      = 10 * time.Second
	defaultPorts    = "0,1,2,3,4,5,6,7"
)

func portsToSlice(ports string) ([]uint32, error) {
	p := strings.Split(ports, ",")
	res := make([]uint32, len(p))
	for idx, vStr := range p {
		v, err := strconv.Atoi(vStr)
		if err != nil {
			return nil, err
		}
		res[idx] = uint32(v)
	}
	return res, nil
}

func initialize(p4RtC *client.Client, ports []uint32) error {
	// generate a digest message for every data plane notification, not appropriate for
	// production
	digestConfig := &p4_v1.DigestEntry_Config{
		MaxTimeoutNs: 0,
		MaxListSize:  1,
		AckTimeoutNs: time.Second.Nanoseconds(),
	}
	log.Debugf("Enabling digest 'digest_t'")
	if err := p4RtC.EnableDigest("digest_t", digestConfig); err != nil {
		return fmt.Errorf("Cannot enable digest 'digest_t': %v", err)
	}

	log.Debugf("Configuring multicast group %d for broadcast", mgrp)
	// TODO: ports should be configurable
	if err := p4RtC.InsertMulticastGroup(mgrp, ports); err != nil {
		return fmt.Errorf("Cannot configure multicast group %d for broadcast: %v", mgrp, err)
	}

	log.Debugf("Setting default action for 'dmac' table to 'broadcast'")
	mgrpBytes, _ := conversion.UInt32ToBinary(mgrp, 2)
	dmacEntry := p4RtC.NewTableEntry("IngressImpl.dmac", "IngressImpl.broadcast", nil, [][]byte{mgrpBytes}, nil)
	if err := p4RtC.ModifyTableEntry(dmacEntry); err != nil {
		return fmt.Errorf("Cannot set default action for 'dmac': %v", err)
	}
	return nil
}

func cleanup(p4RtC *client.Client) error {
	// necessary because of https://github.com/p4lang/behavioral-model/issues/891
	if err := p4RtC.DeleteMulticastGroup(mgrp); err != nil {
		return fmt.Errorf("Cannot delete multicast group %d: %v", err)
	}
	return nil
}

func learnMacs(p4RtC *client.Client, digestList *p4_v1.DigestList) error {
	for _, digestData := range digestList.Data {
		s := digestData.GetStruct()
		srcAddr := s.Members[0].GetBitstring()
		ingressPort := s.Members[1].GetBitstring()
		log.WithFields(log.Fields{
			"srcAddr":     srcAddr,
			"ingressPort": ingressPort,
		}).Debugf("Learning MAC")

		smacOptions := &client.TableEntryOptions{
			IdleTimeout: macTimeout,
		}

		smacEntry := p4RtC.NewTableEntry("IngressImpl.smac", "NoAction", []client.MatchInterface{&client.ExactMatch{srcAddr}}, nil, smacOptions)
		if err := p4RtC.InsertTableEntry(smacEntry); err != nil {
			log.Errorf("Cannot insert entry in 'smac': %v", err)
		}

		dmacEntry := p4RtC.NewTableEntry("IngressImpl.dmac", "IngressImpl.fwd", []client.MatchInterface{&client.ExactMatch{srcAddr}}, [][]byte{ingressPort}, nil)
		if err := p4RtC.InsertTableEntry(dmacEntry); err != nil {
			log.Errorf("Cannot insert entry in 'dmac': %v", err)
		}
	}

	if err := p4RtC.AckDigestList(digestList); err != nil {
		return fmt.Errorf("Error when acking digest list: %v", err)
	}

	return nil
}

func forgetEntries(p4RtC *client.Client, notification *p4_v1.IdleTimeoutNotification) {
	for _, entry := range notification.TableEntry {
		srcAddr := entry.Match[0].GetExact().Value
		log.WithFields(log.Fields{
			"srcAddr": srcAddr,
		}).Debugf("Expiring MAC")

		// first delete from the dmac table, then enable learning again for that MAC by
		// deleting from the smac table.

		dmacEntry := p4RtC.NewTableEntry("IngressImpl.dmac", "", []client.MatchInterface{&client.ExactMatch{srcAddr}}, nil, nil)
		if err := p4RtC.DeleteTableEntry(dmacEntry); err != nil {
			log.Errorf("Cannot delete entry from 'dmac': %v", err)
		}

		if err := p4RtC.DeleteTableEntry(entry); err != nil {
			log.Errorf("Cannot delete entry from 'smac': %v", err)
		}
	}
}

func handleStreamMessages(p4RtC *client.Client, messageCh <-chan *p4_v1.StreamMessageResponse) {
	for message := range messageCh {
		switch m := message.Update.(type) {
		case *p4_v1.StreamMessageResponse_Packet:
			log.Debugf("Received PacketIn")
		case *p4_v1.StreamMessageResponse_Digest:
			log.Debugf("Received DigestList")
			if err := learnMacs(p4RtC, m.Digest); err != nil {
				log.Errorf("Error when learning MACs: %v", err)
			}
		case *p4_v1.StreamMessageResponse_IdleTimeoutNotification:
			log.Debugf("Received IdleTimeoutNotification")
			forgetEntries(p4RtC, m.IdleTimeoutNotification)
		case *p4_v1.StreamMessageResponse_Error:
			log.Errorf("Received StreamError")
		default:
			log.Errorf("Received unknown stream message")
		}
	}
}

func main() {
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
	var switchPorts string
	flag.StringVar(&switchPorts, "ports", defaultPorts, "List of switch ports - required for configuring multicast group for broadcast")

	flag.Parse()

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	ports, err := portsToSlice(switchPorts)
	if err != nil {
		log.Fatalf("Cannot parse port list: %v", err)
	}

	binBytes := MustAsset("cmd/l2_switch/l2_switch.out/l2_switch.json")
	p4infoBytes := MustAsset("cmd/l2_switch/l2_switch.out/p4info.pb.txt")

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
	resp, err := c.Capabilities(context.Background(), &p4_v1.CapabilitiesRequest{})
	if err != nil {
		log.Fatalf("Error in Capabilities RPC: %v", err)
	}
	log.Infof("P4Runtime server version is %s", resp.P4RuntimeApiVersion)

	stopCh := signals.RegisterSignalHandlers()

	electionID := p4_v1.Uint128{High: 0, Low: 1}

	p4RtC := client.NewClient(c, deviceID, electionID)
	mastershipCh := make(chan bool)
	messageCh := make(chan *p4_v1.StreamMessageResponse, 1000)
	defer close(messageCh)
	go p4RtC.Run(stopCh, mastershipCh, messageCh)

	waitCh := make(chan struct{})

	go func() {
		sent := false
		for isMaster := range mastershipCh {
			if isMaster {
				log.Infof("We are master!")
				if !sent {
					waitCh <- struct{}{}
					sent = true
				}
			} else {
				log.Infof("We are not master!")
			}
		}
	}()

	// it would also be safe to spawn multiple goroutines to handle messages from the channel
	go handleStreamMessages(p4RtC, messageCh)

	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-ctx.Done():
		log.Fatalf("Could not acquire mastership within %v", timeout)
	case <-waitCh:
	}

	log.Info("Setting forwarding pipe")
	if err := p4RtC.SetFwdPipeFromBytes(binBytes, p4infoBytes); err != nil {
		log.Fatalf("Error when setting forwarding pipe: %v", err)
	}

	if err := initialize(p4RtC, ports); err != nil {
		log.Fatalf("Error when initializing defaults: %v", err)
	}
	defer func() {
		if err := cleanup(p4RtC); err != nil {
			log.Errorf("Error during cleanup: %v", err)
		}
	}()

	log.Info("Do Ctrl-C to quit")
	<-stopCh
	log.Info("Stopping client")
}
