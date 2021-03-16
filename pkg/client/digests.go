package client

import (
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

func (c *Client) AckDigestList(digestList *p4_v1.DigestList) error {
	m := &p4_v1.StreamMessageRequest{
		Update: &p4_v1.StreamMessageRequest_DigestAck{DigestAck: &p4_v1.DigestListAck{
			DigestId: digestList.DigestId,
			ListId:   digestList.ListId,
		}},
	}
	c.streamSendCh <- m
	return nil
}

func (c *Client) EnableDigest(digest string, config *p4_v1.DigestEntry_Config) error {
	digestID := c.digestId(digest)
	entry := &p4_v1.DigestEntry{
		DigestId: digestID,
		Config:   config,
	}
	update := &p4_v1.Update{
		Type: p4_v1.Update_INSERT,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_DigestEntry{DigestEntry: entry},
		},
	}
	return c.WriteUpdate(update)
}

func (c *Client) ModifyDigest(digest string, config *p4_v1.DigestEntry_Config) error {
	digestID := c.digestId(digest)
	entry := &p4_v1.DigestEntry{
		DigestId: digestID,
		Config:   config,
	}
	update := &p4_v1.Update{
		Type: p4_v1.Update_MODIFY,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_DigestEntry{DigestEntry: entry},
		},
	}
	return c.WriteUpdate(update)
}

func (c *Client) DisableDigest(digest string) error {
	digestID := c.digestId(digest)
	entry := &p4_v1.DigestEntry{
		DigestId: digestID,
	}
	update := &p4_v1.Update{
		Type: p4_v1.Update_DELETE,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_DigestEntry{DigestEntry: entry},
		},
	}
	return c.WriteUpdate(update)
}
