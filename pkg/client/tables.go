package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"

	"github.com/antoninbas/p4runtime-go-client/pkg/util/conversion"
)

const (
	tableWildcardReadChSize = 100
)

func ToCanonicalIf(v []byte, cond bool) []byte {
	if cond {
		return conversion.ToCanonicalBytestring(v)
	} else {
		return v
	}
}

type MatchInterface interface {
	get(ID uint32, canonical bool) *p4_v1.FieldMatch
}

type ExactMatch struct {
	Value []byte
}

func (m *ExactMatch) get(ID uint32, canonical bool) *p4_v1.FieldMatch {
	exact := &p4_v1.FieldMatch_Exact{
		Value: ToCanonicalIf(m.Value, canonical),
	}
	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Exact_{Exact: exact},
	}
	return mf
}

type LpmMatch struct {
	Value []byte
	PLen  int32
}

func (m *LpmMatch) get(ID uint32, canonical bool) *p4_v1.FieldMatch {
	lpm := &p4_v1.FieldMatch_LPM{
		Value:     m.Value,
		PrefixLen: m.PLen,
	}

	// P4Runtime has strict rules regarding ternary matches: in the case of
	// LPM, trailing bits in the value (after prefix) must be set to 0.
	firstByteMasked := int(m.PLen / 8)
	if firstByteMasked != len(lpm.Value) {
		i := firstByteMasked
		r := m.PLen % 8
		lpm.Value[i] = lpm.Value[i] & (0xff << (8 - r))
		for i = i + 1; i < len(lpm.Value); i++ {
			lpm.Value[i] = 0
		}
	}

	lpm.Value = ToCanonicalIf(lpm.Value, canonical)

	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Lpm{Lpm: lpm},
	}
	return mf
}

type TernaryMatch struct {
	Value []byte
	Mask  []byte
}

func (m *TernaryMatch) get(ID uint32, canonical bool) *p4_v1.FieldMatch {
	ternary := &p4_v1.FieldMatch_Ternary{
		Value: m.Value,
		Mask:  m.Mask,
	}

	// P4Runtime has strict rules regarding ternary matches: masked off bits
	// must be set to 0 in the value.
	offset := len(ternary.Mask) - len(ternary.Value)
	if offset < 0 {
		ternary.Value = ternary.Value[-offset:]
		offset = 0
	}

	for i, b := range ternary.Mask[offset:] {
		ternary.Value[i] = ternary.Value[i] & b
	}

	ternary.Value = ToCanonicalIf(ternary.Value, canonical)
	ternary.Mask = ToCanonicalIf(ternary.Mask, canonical)

	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Ternary_{Ternary: ternary},
	}
	return mf
}

type RangeMatch struct {
	Low  []byte
	High []byte
}

func (m *RangeMatch) get(ID uint32, canonical bool) *p4_v1.FieldMatch {
	fmRange := &p4_v1.FieldMatch_Range{
		Low:  ToCanonicalIf(m.Low, canonical),
		High: ToCanonicalIf(m.High, canonical),
	}

	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Range_{Range: fmRange},
	}
	return mf
}

type OptionalMatch struct {
	Value []byte
}

func (m *OptionalMatch) get(ID uint32, canonical bool) *p4_v1.FieldMatch {
	optional := &p4_v1.FieldMatch_Optional{
		Value: ToCanonicalIf(m.Value, canonical),
	}

	mf := &p4_v1.FieldMatch{
		FieldId:        ID,
		FieldMatchType: &p4_v1.FieldMatch_Optional_{Optional: optional},
	}
	return mf
}

type TableEntryOptions struct {
	IdleTimeout time.Duration
	Priority    int32
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
		Type: &p4_v1.TableAction_Action{Action: c.newAction(action, params)},
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
			Type: &p4_v1.TableAction_ActionProfileActionSet{
				ActionProfileActionSet: &p4_v1.ActionProfileActionSet{},
			},
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
			Action: s.client.newAction(action, params),
			Weight: weight,
			WatchKind: &p4_v1.ActionProfileAction_WatchPort{
				WatchPort: port.AsBytes(),
			},
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
		TableId: tableID,
		//nolint:staticcheck // SA5011 if mfs==nil then for loop is not executed by default
		IsDefaultAction: (mfs == nil),
		Action:          action,
	}

	//nolint:staticcheck // SA5011 if mfs==nil then for loop is not executed by default
	//lint:ignore SA5011 This line added for support golint version of VSC
	for idx, mf := range mfs {
		entry.Match = append(entry.Match, mf.get(uint32(idx+1), c.CanonicalBytestrings))
	}

	if options != nil {
		entry.IdleTimeoutNs = options.IdleTimeout.Nanoseconds()
		entry.Priority = options.Priority
	}

	return entry
}

func (c *Client) ReadTableEntry(ctx context.Context, table string, mfs []MatchInterface) (*p4_v1.TableEntry, error) {
	tableID := c.tableId(table)

	entry := &p4_v1.TableEntry{
		TableId: tableID,
	}

	for idx, mf := range mfs {
		entry.Match = append(entry.Match, mf.get(uint32(idx+1), c.CanonicalBytestrings))
	}

	entity := &p4_v1.Entity{
		Entity: &p4_v1.Entity_TableEntry{TableEntry: entry},
	}

	readEntity, err := c.ReadEntitySingle(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("error when reading table entry: %v", err)
	}

	readEntry := readEntity.GetTableEntry()
	if readEntry == nil {
		return nil, fmt.Errorf("server returned an entity but it is not a table entry! ")
	}

	return readEntry, nil
}

func (c *Client) ReadTableEntryWildcard(ctx context.Context, table string) ([]*p4_v1.TableEntry, error) {
	tableID := c.tableId(table)

	entry := &p4_v1.TableEntry{
		TableId: tableID,
	}

	out := make([]*p4_v1.TableEntry, 0)
	readEntityCh := make(chan *p4_v1.Entity, tableWildcardReadChSize)

	var wg sync.WaitGroup
	var err error
	wg.Add(1)

	go func() {
		defer wg.Done()
		for readEntity := range readEntityCh {
			readEntry := readEntity.GetTableEntry()
			if readEntry != nil {
				out = append(out, readEntry)
			} else if err == nil {
				// only set the error if this is the first error we encounter
				// dp not stop reading from the channel, as doing so would cause
				// ReadEntityWildcard to block indefinitely
				err = fmt.Errorf("server returned an entity which is not a table entry!")
			}
		}
	}()

	if err := c.ReadEntityWildcard(ctx, &p4_v1.Entity{
		Entity: &p4_v1.Entity_TableEntry{TableEntry: entry},
	}, readEntityCh); err != nil {
		return nil, fmt.Errorf("error when reading table entries: %v", err)
	}

	wg.Wait()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) InsertTableEntry(ctx context.Context, entry *p4_v1.TableEntry) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_INSERT,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_TableEntry{TableEntry: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) ModifyTableEntry(ctx context.Context, entry *p4_v1.TableEntry) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_MODIFY,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_TableEntry{TableEntry: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}

func (c *Client) DeleteTableEntry(ctx context.Context, entry *p4_v1.TableEntry) error {
	update := &p4_v1.Update{
		Type: p4_v1.Update_DELETE,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_TableEntry{TableEntry: entry},
		},
	}

	return c.WriteUpdate(ctx, update)
}
