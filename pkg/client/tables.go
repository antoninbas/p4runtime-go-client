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

type TernaryMatch struct {
	Value []byte
	Mask  []byte
}

func (m *TernaryMatch) get(ID uint32) *p4_v1.FieldMatch {
	ternary := &p4_v1.FieldMatch_Ternary{
		Value: m.Value,
		Mask:  m.Mask,
	}
	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Ternary_{ternary},
	}
	return mf
}

type RangeMatch struct {
	Low  []byte
	High []byte
}

func (m *RangeMatch) get(ID uint32) *p4_v1.FieldMatch {
	fmRange := &p4_v1.FieldMatch_Range{
		Low:  m.Low,
		High: m.High,
	}

	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Range_{fmRange},
	}
	return mf
}

type OptionalMatch struct {
	Value []byte
}

func (m *OptionalMatch) get(ID uint32) *p4_v1.FieldMatch {
	optional := &p4_v1.FieldMatch_Optional{
		Value: m.Value,
	}

	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Optional_{optional},
	}
	return mf
}

type TableEntryOptions struct {
	IdleTimeout time.Duration
}

func (c *Client) newAction(action string, params [][]byte) *p4_v1.Action {
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

	return directAction
}

func (c *Client) NewTableActionDirect(
	action string,
	params [][]byte,
) *p4_v1.TableAction {
	return &p4_v1.TableAction{
		Type: &p4_v1.TableAction_Action{c.newAction(action, params)},
	}
}

type ActionProfileActionSet struct {
	client *Client
	action *p4_v1.TableAction
}

func (c *Client) NewActionProfileActionSet() *ActionProfileActionSet {
	return &ActionProfileActionSet{
		client: c,
		action: &p4_v1.TableAction{
			Type: &p4_v1.TableAction_ActionProfileActionSet{&p4_v1.ActionProfileActionSet{}},
		},
	}
}

func (s *ActionProfileActionSet) AddAction(
	action string,
	params [][]byte,
	weight int32,
	port Port,
) *ActionProfileActionSet {
	actionSet := s.action.GetActionProfileActionSet()
	actionSet.ActionProfileActions = append(
		actionSet.ActionProfileActions,
		&p4_v1.ActionProfileAction{
			Action:    s.client.newAction(action, params),
			Weight:    weight,
			WatchKind: &p4_v1.ActionProfileAction_WatchPort{port.AsBytes()},
		},
	)
	return s
}

func (s *ActionProfileActionSet) TableAction() *p4_v1.TableAction {
	return s.action
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
