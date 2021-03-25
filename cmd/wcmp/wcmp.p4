#define V1MODEL_VERSION 20200408

#include <core.p4>
#include <v1model.p4>

typedef bit<48> MacAddr_t;
typedef bit<32> IPv4_t;

enum bit<16> EtherType_t {
  IPv4      = 0x0800
}

header ethernet_t {
    MacAddr_t dstAddr;
    MacAddr_t srcAddr;
    EtherType_t etherType;
}

header ipv4_t {
    bit<4>  version;
    bit<4>  ihl;
    bit<8>  diffserv;
    bit<16> totalLen;
    bit<16> identification;
    bit<3>  flags;
    bit<13> fragOffset;
    bit<8>  ttl;
    bit<8>  protocol;
    bit<16> hdrChecksum;
    IPv4_t srcAddr;
    IPv4_t dstAddr;
}

typedef bit<256> StringId_t;

struct metadata {
    StringId_t group_id;
    StringId_t nhop_id;
}

struct headers {
    ethernet_t ethernet;
    ipv4_t ipv4;
}

parser ParserImpl(packet_in packet, out headers hdr, inout metadata meta, inout standard_metadata_t standard_metadata) {
    state parse_ethernet {
        packet.extract(hdr.ethernet);
        transition select(hdr.ethernet.etherType) {
            EtherType_t.IPv4: parse_ipv4;
            _: accept;
        }
    }
    state parse_ipv4 {
        packet.extract(hdr.ipv4);
        transition accept;
    }
    state start {
        transition parse_ethernet;
    }
}

action drop(inout standard_metadata_t standard_metadata) {
    mark_to_drop(standard_metadata);
}

control IngressImpl(inout headers hdr, inout metadata meta, inout standard_metadata_t standard_metadata) {
    action set_group(StringId_t group_id) {
        hdr.ipv4.ttl = hdr.ipv4.ttl - 1;
        meta.group_id = group_id;
    }
    action set_nhop(StringId_t nhop_id) {
        meta.nhop_id = nhop_id;
    }
    action set_eg_port(PortId_t eg_port) {
        standard_metadata.egress_spec = eg_port;
    }
    table l3_lpm {
        key = {
            hdr.ipv4.dstAddr : lpm;
        }
        actions = {
            drop(standard_metadata);
            set_group;
        }
        const default_action = drop(standard_metadata);
        size = 131072;
    }
    @max_group_size(256)
    action_selector(HashAlgorithm.crc32, 65536, 32w16) wcmp_group_selector;
    table wcmp_group {
        key = {
            meta.group_id : exact;
        }
        actions = {
            drop(standard_metadata);
            set_nhop;
        }
        implementation = wcmp_group_selector;
        const default_action = drop(standard_metadata);
        size = 4096;
    }
    table nhop {
        key = {
            meta.nhop_id : exact;
        }
        actions = {
            drop(standard_metadata);
            set_eg_port;
        }
        const default_action = drop(standard_metadata);
        size = 16384;
    }
    apply {
        if (!hdr.ipv4.isValid()) {
            log_msg("Dropping non-ipv4 packet");
            exit;
        }
        if (standard_metadata.checksum_error == 1) {
            log_msg("Dropping ipv4 packet with invalid checksum");
            exit;
        }
        if (l3_lpm.apply().hit) {
            if (wcmp_group.apply().hit) {
                nhop.apply();
            }
        }
    }
}

control EgressImpl(inout headers hdr, inout metadata meta, inout standard_metadata_t standard_metadata) {
    action do_rewrites(MacAddr_t srcAddr, MacAddr_t dstAddr) {
        hdr.ethernet.srcAddr = srcAddr;
        hdr.ethernet.dstAddr = dstAddr;
    }
    table rewrites {
        key = {
            meta.nhop_id : exact;
        }
        actions = {
            drop(standard_metadata);
            do_rewrites;
        }
        const default_action = drop(standard_metadata);
        size = 16384;
    }
    apply {
        rewrites.apply();
    }
}

control DeparserImpl(packet_out packet, in headers hdr) {
    apply {
        packet.emit(hdr.ethernet);
        packet.emit(hdr.ipv4);
    }
}

control verifyChecksum(inout headers hdr, inout metadata meta) {
    apply {
        verify_checksum(
            hdr.ipv4.isValid(),
            {hdr.ipv4.version, hdr.ipv4.ihl, hdr.ipv4.diffserv,
             hdr.ipv4.totalLen, hdr.ipv4.identification, hdr.ipv4.flags,
             hdr.ipv4.fragOffset, hdr.ipv4.ttl, hdr.ipv4.protocol,
             hdr.ipv4.srcAddr, hdr.ipv4.dstAddr},
            hdr.ipv4.hdrChecksum, HashAlgorithm.csum16
        );
    }
}

control computeChecksum(inout headers hdr, inout metadata meta) {
    apply {
        update_checksum(
            hdr.ipv4.isValid(),
            {hdr.ipv4.version, hdr.ipv4.ihl, hdr.ipv4.diffserv,
             hdr.ipv4.totalLen, hdr.ipv4.identification, hdr.ipv4.flags,
             hdr.ipv4.fragOffset, hdr.ipv4.ttl, hdr.ipv4.protocol,
             hdr.ipv4.srcAddr, hdr.ipv4.dstAddr},
            hdr.ipv4.hdrChecksum, HashAlgorithm.csum16
        );
    }
}

V1Switch(p = ParserImpl(),
         ig = IngressImpl(),
         vr = verifyChecksum(),
         eg = EgressImpl(),
         ck = computeChecksum(),
         dep = DeparserImpl()) main;
