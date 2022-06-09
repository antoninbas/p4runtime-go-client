package client

import (
	"context"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

func (c *Client) NewTableActionMember(
	memberID uint32,
) *p4_v1.TableAction {
	return &p4_v1.TableAction{
		Type: &p4_v1.TableAction_ActionProfileMemberId{ActionProfileMemberId: memberID},
	}
}

func (c *Client) NewTableActionGroup(
	groupID uint32,
) *p4_v1.TableAction {
	return &p4_v1.TableAction{
		Type: &p4_v1.TableAction_ActionProfileGroupId{ActionProfileGroupId: groupID},
	}
}

func (c *Client) NewActionProfileMember(
	table string,
	memberID uint32,
	action string,
	params [][]byte,
) *p4_v1.ActionProfileMember {
	tableID := c.actionProfileId(table)

	entry := &p4_v1.ActionProfileMember{
		ActionProfileId: tableID,
		MemberId: memberID,
		Action: c.newAction(action, params),
	}

	return entry
}

func (c *Client) InsertActionProfileMember(ctx context.Context, entry *p4_v1.ActionProfileMember) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_INSERT,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_ActionProfileMember{ActionProfileMember: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) ModifyActionProfileMember(ctx context.Context, entry *p4_v1.ActionProfileMember) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_MODIFY,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_ActionProfileMember{ActionProfileMember: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) DeleteActionProfileMember(ctx context.Context, entry *p4_v1.ActionProfileMember) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_DELETE,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_ActionProfileMember{ActionProfileMember: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) NewActionProfileGroup(
	table string,
	groupID uint32,
	members []*p4_v1.ActionProfileGroup_Member,
	size int32,
) *p4_v1.ActionProfileGroup {
	tableID := c.actionProfileId(table)

	entry := &p4_v1.ActionProfileGroup{
		ActionProfileId: tableID,
		GroupId: groupID,
		Members: members,
		MaxSize: size,
	}

	return entry
}

func (c *Client) InsertActionProfileGroup(ctx context.Context, entry *p4_v1.ActionProfileGroup) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_INSERT,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_ActionProfileGroup{ActionProfileGroup: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) ModifyActionProfileGroup(ctx context.Context, entry *p4_v1.ActionProfileGroup) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_MODIFY,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_ActionProfileGroup{ActionProfileGroup: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) DeleteActionProfileGroup(ctx context.Context, entry *p4_v1.ActionProfileGroup) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_DELETE,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_ActionProfileGroup{ActionProfileGroup: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}
