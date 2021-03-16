package client

import (
	"fmt"
	"sync"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

func (c *Client) ModifyCounterEntry(counter string, index int64, data *p4_v1.CounterData) error {
	counterID := c.counterId(counter)
	entry := &p4_v1.CounterEntry{
		CounterId: counterID,
		Index:     &p4_v1.Index{Index: index},
		Data:      data,
	}
	update := &p4_v1.Update{
		Type: p4_v1.Update_MODIFY,
		Entity: &p4_v1.Entity{
			Entity: &p4_v1.Entity_CounterEntry{CounterEntry: entry},
		},
	}
	return c.WriteUpdate(update)
}

func (c *Client) ReadCounterEntry(counter string, index int64) (*p4_v1.CounterData, error) {
	counterID := c.counterId(counter)
	entry := &p4_v1.CounterEntry{
		CounterId: counterID,
		Index:     &p4_v1.Index{Index: index},
	}
	readEntity, err := c.ReadEntitySingle(&p4_v1.Entity{
		Entity: &p4_v1.Entity_CounterEntry{CounterEntry: entry},
	})
	if err != nil {
		return nil, fmt.Errorf("error when reading counter entry: %v", err)
	}
	readEntry := readEntity.GetCounterEntry()
	if readEntry == nil {
		return nil, fmt.Errorf("server returned an entity but it is not a counter entry!")
	}
	return readEntry.Data, nil
}

func (c *Client) ReadCounterEntryWildcard(counter string) ([]*p4_v1.CounterData, error) {
	p4Counter := c.findCounter(counter)
	entry := &p4_v1.CounterEntry{
		CounterId: p4Counter.Preamble.Id,
	}
	out := make([]*p4_v1.CounterData, 0, p4Counter.Size)
	readEntityCh := make(chan *p4_v1.Entity, 100)
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func() {
		defer wg.Done()
		for readEntity := range readEntityCh {
			readEntry := readEntity.GetCounterEntry()
			if readEntry == nil {
				err = fmt.Errorf("server returned an entity which is not a counter entry!")
			}
			out = append(out, readEntry.Data)
		}
	}()
	if err := c.ReadEntityWildcard(&p4_v1.Entity{
		Entity: &p4_v1.Entity_CounterEntry{CounterEntry: entry},
	}, readEntityCh); err != nil {
		return nil, fmt.Errorf("error when reading counter entries: %v", err)
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}
	return out, nil
}
