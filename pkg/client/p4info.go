package client

import (
	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
)

const invalidID = 0

func (c *Client) tableId(name string) uint32 {
	if c.p4Info == nil {
		return invalidID
	}
	for _, table := range c.p4Info.Tables {
		if table.Preamble.Name == name {
			return table.Preamble.Id
		}
	}
	return invalidID
}

func (c *Client) findTable(name string) *p4_config_v1.Table {
	if c.p4Info == nil {
		return nil
	}
	for _, table := range c.p4Info.Tables {
		if table.Preamble.Name == name {
			return table
		}
	}
	return nil
}

func (c *Client) matchFieldId(tableName, fieldName string) uint32 {
	if c.p4Info == nil {
		return invalidID
	}
	table := c.findTable(tableName)
	if table == nil {
		return invalidID
	}
	for _, mf := range table.MatchFields {
		if mf.Name == fieldName {
			return mf.Id
		}
	}
	return invalidID
}

func (c *Client) actionId(name string) uint32 {
	if c.p4Info == nil {
		return invalidID
	}
	for _, action := range c.p4Info.Actions {
		if action.Preamble.Name == name {
			return action.Preamble.Id
		}
	}
	return invalidID
}

func (c *Client) actionProfileId(name string) uint32 {
	if c.p4Info == nil {
		return invalidID
	}
	for _, actionProfile := range c.p4Info.ActionProfiles {
		if actionProfile.Preamble.Name == name {
			return actionProfile.Preamble.Id
		}
	}
	return invalidID
}

func (c *Client) digestId(name string) uint32 {
	if c.p4Info == nil {
		return invalidID
	}
	for _, digest := range c.p4Info.Digests {
		if digest.Preamble.Name == name {
			return digest.Preamble.Id
		}
	}
	return invalidID
}

func (c *Client) findCounter(name string) *p4_config_v1.Counter {
	if c.p4Info == nil {
		return nil
	}
	for _, counter := range c.p4Info.Counters {
		if counter.Preamble.Name == name {
			return counter
		}
	}
	return nil
}

func (c *Client) counterId(name string) uint32 {
	counter := c.findCounter(name)
	if counter == nil {
		return invalidID
	}
	return counter.Preamble.Id
}

func (c *Client) findMeter(name string) *p4_config_v1.Meter {
	if c.p4Info == nil {
		return nil
	}
	for _, meter := range c.p4Info.Meters {
		if meter.Preamble.Name == name {
			return meter
		}
	}
	return nil
}

func (c *Client) meterId(name string) uint32 {
	meter := c.findMeter(name)
	if meter == nil {
		return invalidID
	}
	return meter.Preamble.Id
}
