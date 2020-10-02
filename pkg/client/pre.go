package client

import (
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

func (c *Client) InsertMulticastGroup(mgid uint32, ports []uint32) error {
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
			entry,
		},
	}

	var updateType p4_v1.Update_Type
	updateType = p4_v1.Update_INSERT
	update := &p4_v1.Update{
		Type: updateType,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_PacketReplicationEngineEntry{preEntry},
		},
	}

	return c.WriteUpdate(update)
}

func (c *Client) DeleteMulticastGroup(mgid uint32) error {
	entry := &p4_v1.MulticastGroupEntry{
		MulticastGroupId: mgid,
	}

	preEntry := &p4_v1.PacketReplicationEngineEntry{
		Type: &p4_v1.PacketReplicationEngineEntry_MulticastGroupEntry{
			entry,
		},
	}

	var updateType p4_v1.Update_Type
	updateType = p4_v1.Update_DELETE
	update := &p4_v1.Update{
		Type: updateType,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_PacketReplicationEngineEntry{preEntry},
		},
	}

	return c.WriteUpdate(update)
}
