package client

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
