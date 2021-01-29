package client

import (
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

type MatchInterface interface {
	get(ID uint32) *p4_v1.FieldMatch
}

type ExactMatch struct {
	Value []byte
}

func (m *ExactMatch) get(ID uint32) *p4_v1.FieldMatch {
	exact := &p4_v1.FieldMatch_Exact{
		Value: m.Value,
	}
	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Exact_{exact},
	}
	return mf
}

type LpmMatch struct {
	Value []byte
	PLen  int32
}

func (m *LpmMatch) get(ID uint32) *p4_v1.FieldMatch {
	lpm := &p4_v1.FieldMatch_LPM{
		Value:     m.Value,
		PrefixLen: m.PLen,
	}

	// P4Runtime now has strict rules regarding ternary matches: in the
	// case of LPM, trailing bits in the value (after prefix) must be set
	// to 0.
	firstByteMasked := int(m.PLen / 8)
	if firstByteMasked != len(m.Value) {
		i := firstByteMasked
		r := m.PLen % 8
		m.Value[i] = m.Value[i] & (0xff << (8 - r))
		for i = i + 1; i < len(m.Value); i++ {
			m.Value[i] = 0
		}
	}

	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Lpm{lpm},
	}
	return mf
}

type TableEntryOptions struct {
	IdleTimeout time.Duration
}

func (c *Client) NewTableActionDirect(
	action string,
	params [][]byte,
) *p4_v1.TableAction {
	actionID := c.actionId(action)
	directAction := &p4_v1.Action{
		ActionId: actionID,
	}

	for idx, p := range params {
		param := &p4_v1.Action_Param{
			ParamId: uint32(idx + 1),
			Value:   p,
		}
		directAction.Params = append(directAction.Params, param)
	}

	return &p4_v1.TableAction{
		Type: &p4_v1.TableAction_Action{directAction},
	}
}

// for default entries: to set use nil for mfs, to unset use nil for mfs and nil
// for action
func (c *Client) NewTableEntry(
	table string,
	mfs []MatchInterface,
	action *p4_v1.TableAction,
	options *TableEntryOptions,
) *p4_v1.TableEntry {
	tableID := c.tableId(table)

	entry := &p4_v1.TableEntry{
		TableId:         tableID,
		IsDefaultAction: (mfs == nil),
		Action:          action,
	}

	for idx, mf := range mfs {
		entry.Match = append(entry.Match, mf.get(uint32(idx+1)))
	}

	if options != nil {
		entry.IdleTimeoutNs = options.IdleTimeout.Nanoseconds()
	}

	return entry
}

func (c *Client) InsertTableEntry(entry *p4_v1.TableEntry) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_INSERT,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_TableEntry{entry},
		},
	}

	return c.WriteUpdate(update)
}

func (c *Client) ModifyTableEntry(entry *p4_v1.TableEntry) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_MODIFY,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_TableEntry{entry},
		},
	}

	return c.WriteUpdate(update)
}

func (c *Client) DeleteTableEntry(entry *p4_v1.TableEntry) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_DELETE,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_TableEntry{entry},
		},
	}

	return c.WriteUpdate(update)
}
