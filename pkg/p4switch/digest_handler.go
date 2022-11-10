package p4switch

import (
	"context"
	"github.com/antoninbas/p4runtime-go-client/pkg/util/conversion"
	"fmt"
	"net"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

var digestConfig p4_v1.DigestEntry_Config = p4_v1.DigestEntry_Config{
	MaxTimeoutNs: 0,
	MaxListSize:  1,
	AckTimeoutNs: time.Second.Nanoseconds() * 1000,
}

func (sw *GrpcSwitch) EnableDigest(ctx context.Context) error {
	digestName := sw.GetDigests()
	for _, digest := range digestName {
		if digest == "" {
			continue
		}
		if err := sw.p4RtC.EnableDigest(ctx,digest, &digestConfig); err != nil {
			return fmt.Errorf("cannot enable digest %s", digest)
		}
		sw.log.Debugf("Enabled digest %s", digest)
	}
	return nil
}

/* --- CHANGE FROM NOW ON --- */

type digest_t struct {
	srcAddr  net.IP
	dstAddr  net.IP
	srcPort  int
	dstPort  int
	pktCount uint64
}

func (sw *GrpcSwitch) HandleDigest(ctx context.Context,digestList *p4_v1.DigestList) {
	for _, digestData := range digestList.Data {
		digestStruct := parseDigestData(digestData.GetStruct())
		sw.log.Debugf("%s P%d -> %s P%d pkt %d", digestStruct.srcAddr, digestStruct.srcPort, digestStruct.dstAddr, digestStruct.dstPort, digestStruct.pktCount)
	}
	if err := sw.p4RtC.AckDigestList(ctx,digestList); err != nil {
		sw.errCh <- err
	}
	sw.log.Trace("Ack digest list")
}

func parseDigestData(str *p4_v1.P4StructLike) digest_t {
	srcAddrByte := str.Members[0].GetBitstring()
	dstAddrByte := str.Members[1].GetBitstring()
	srcAddr := conversion.BinaryToIpv4(srcAddrByte)
	dstAddr := conversion.BinaryToIpv4(dstAddrByte)
	srcPort := conversion.BinaryCompressedToUint16(str.Members[2].GetBitstring())
	dstPort := conversion.BinaryCompressedToUint16(str.Members[3].GetBitstring())
	pktCount := conversion.BinaryCompressedToUint64(str.Members[4].GetBitstring())
	return digest_t{
		srcAddr:  srcAddr,
		dstAddr:  dstAddr,
		srcPort:  int(srcPort),
		dstPort:  int(dstPort),
		pktCount: pktCount,
	}
}
