package client

import (
	"context"
	"fmt"
	"sync"

	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

const (
	meterWildcardReadChSize = 100
)

func (c *Client) ReadMeterEntry(ctx context.Context, meter string, index int64) (*p4_v1.MeterConfig, error) {
	meterID := c.meterId(meter)
	if meterID == invalidID {
		return nil, fmt.Errorf("meter %s not found", meter)
	}
	entry := &p4_v1.MeterEntry{
		MeterId: meterID,
		Index:   &p4_v1.Index{Index: index},
	}
	readEntity, err := c.ReadEntitySingle(ctx, &p4_v1.Entity{
		Entity: &p4_v1.Entity_MeterEntry{MeterEntry: entry},
	})
	if err != nil {
		return nil, fmt.Errorf("error when reading meter entry: %v", err)
	}
	readEntry := readEntity.GetMeterEntry()
	if readEntry == nil {
		return nil, fmt.Errorf("server returned an entity but it is not a meter entry! ")
	}
	return readEntry.Config, nil
}

func (c *Client) ReadMeterEntryWildcard(ctx context.Context, meter string) ([]*p4_v1.MeterEntry, error) {
	p4Meter := c.findMeter(meter)
	if p4Meter == nil {
		return nil, fmt.Errorf("meter %s not found", meter)
	}
	entry := &p4_v1.MeterEntry{
		MeterId: p4Meter.Preamble.Id,
	}
	out := make([]*p4_v1.MeterEntry, 0, p4Meter.Size)
	readEntityCh := make(chan *p4_v1.Entity, meterWildcardReadChSize)
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func() {
		defer wg.Done()
		for readEntity := range readEntityCh {
			readEntry := readEntity.GetMeterEntry()
			if readEntry != nil {
				out = append(out, readEntry)
			} else if err == nil {
				// only set the error if this is the first error we encounter
				// do not stop reading from the channel, as doing so would cause
				// ReadEntityWildcard to block indefinitely
				err = fmt.Errorf("server returned an entity which is not a meter entry")
			}
		}
	}()
	if err := c.ReadEntityWildcard(ctx, &p4_v1.Entity{
		Entity: &p4_v1.Entity_MeterEntry{MeterEntry: entry},
	}, readEntityCh); err != nil {
		return nil, fmt.Errorf("error when reading meter entries: %v", err)
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}
	return out, nil
}
