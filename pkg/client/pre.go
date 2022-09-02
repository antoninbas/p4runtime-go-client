package client

import (
	"context"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

type CloneSessionOptions struct {
	// ClassOfService defines class_of_service that should be set for cloned packets.
	ClassOfService uint32
	// PacketLenBytes defines packet size (in bytes) for cloned packets.
	// Should be set to non-zero value if packets should be truncated.
	PacketLenBytes int32
}

var DefaultCloneSessionOptions = CloneSessionOptions{
	ClassOfService: 0,
	PacketLenBytes: 0,
}

func (c *Client) InsertCloneSession(ctx context.Context, id uint32, ports []uint32, options CloneSessionOptions) error {
	entry := &p4_v1.CloneSessionEntry{
		SessionId:         id,
		ClassOfService:    options.ClassOfService,
		PacketLengthBytes: options.PacketLenBytes,
	}

	for idx, port := range ports {
		replica := &p4_v1.Replica{
			EgressPort: port,
			Instance:   uint32(idx),
		}
		entry.Replicas = append(entry.Replicas, replica)
	}

	preEntry := &p4_v1.PacketReplicationEngineEntry{
		Type: &p4_v1.PacketReplicationEngineEntry_CloneSessionEntry{
			CloneSessionEntry: entry,
		},
	}

	updateType := p4_v1.Update_INSERT
	update := &p4_v1.Update{
		Type: updateType,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_PacketReplicationEngineEntry{
				PacketReplicationEngineEntry: preEntry,
			},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) DeleteCloneSession(ctx context.Context, id uint32) error {
	entry := &p4_v1.CloneSessionEntry{
		SessionId: id,
	}

	preEntry := &p4_v1.PacketReplicationEngineEntry{
		Type: &p4_v1.PacketReplicationEngineEntry_CloneSessionEntry{
			CloneSessionEntry: entry,
		},
	}

	updateType := p4_v1.Update_DELETE
	update := &p4_v1.Update{
		Type: updateType,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_PacketReplicationEngineEntry{
				PacketReplicationEngineEntry: preEntry,
			},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) InsertMulticastGroup(ctx context.Context, mgid uint32, ports []uint32) error {
	entry := &p4_v1.MulticastGroupEntry{
		MulticastGroupId: mgid,
	}
	for idx, port := range ports {
		replica := &p4_v1.Replica{
			EgressPort: port,
			Instance:   uint32(idx),
		}
		entry.Replicas = append(entry.Replicas, replica)
	}

	preEntry := &p4_v1.PacketReplicationEngineEntry{
		Type: &p4_v1.PacketReplicationEngineEntry_MulticastGroupEntry{
			MulticastGroupEntry: entry,
		},
	}

	updateType := p4_v1.Update_INSERT
	update := &p4_v1.Update{
		Type: updateType,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_PacketReplicationEngineEntry{
				PacketReplicationEngineEntry: preEntry,
			},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) DeleteMulticastGroup(ctx context.Context, mgid uint32) error {
	entry := &p4_v1.MulticastGroupEntry{
		MulticastGroupId: mgid,
	}

	preEntry := &p4_v1.PacketReplicationEngineEntry{
		Type: &p4_v1.PacketReplicationEngineEntry_MulticastGroupEntry{
			MulticastGroupEntry: entry,
		},
	}

	updateType := p4_v1.Update_DELETE
	update := &p4_v1.Update{
		Type: updateType,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_PacketReplicationEngineEntry{
				PacketReplicationEngineEntry: preEntry,
			},
		},
	}

	return c.WriteUpdate(ctx, update)
}
