package client

import(
	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
)


func (c *Client) GetActions() []*p4_config_v1.Action {
	return c.p4Info.Actions
}

func (c *Client) GetTables() []*p4_config_v1.Table {
	return c.p4Info.Tables
}

