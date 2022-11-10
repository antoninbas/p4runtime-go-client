package p4switch

import (
	"github.com/antoninbas/p4runtime-go-client/pkg/client"
	"strconv"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	log "github.com/sirupsen/logrus"
)

type GrpcSwitch struct {
	id                uint64
	initialConfigName string
	config            *SwitchConfig
	ports             int
	addr              string
	log               *log.Entry
	errCh             chan error
	certFile          string
	p4RtC             *client.Client
	messageCh         chan *p4_v1.StreamMessageResponse
}

func (sw *GrpcSwitch) GetName() string {
	return "s" + strconv.FormatUint(sw.id, 10)
}

func (sw *GrpcSwitch) GetLogger() *log.Entry {
	return sw.log
}
