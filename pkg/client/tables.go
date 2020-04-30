package client

import (
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

func (c *Client) InsertTableEntry(table string, action string, mfs []MatchInterface, params [][]byte) error {
	tableID := c.tableId(table)
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

	tableAction := &p4_v1.TableAction{
		Type: &p4_v1.TableAction_Action{directAction},
	}

	entry := &p4_v1.TableEntry{
		TableId:         tableID,
		Action:          tableAction,
		IsDefaultAction: (mfs == nil),
	}

	for idx, mf := range mfs {
		entry.Match = append(entry.Match, mf.get(uint32(idx+1)))
	}

	var updateType p4_v1.Update_Type
	if mfs == nil {
		updateType = p4_v1.Update_MODIFY
	} else {
		updateType = p4_v1.Update_INSERT
	}
	update := &p4_v1.Update{
		Type: updateType,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_TableEntry{entry},
		},
	}

	return c.WriteUpdate(update)
}
