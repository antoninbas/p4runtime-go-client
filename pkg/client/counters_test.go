package client

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

// TestReadCounterEntryWildcard_BadEntity ensures that if the server returns an unexpected entity
// (in our case a TableEntry instead of a CounterEntry), ReadCounterEntryWildcard will fail with an
// error instead of blocking indefinitely.
func TestReadCounterEntryWildcard_BadEntity(t *testing.T) {
	counterName := "testCounter"
	counterId := uint32(100)
	count := 0
	// we need the number of entities following the "bad" entity to exceed the channel capacity
	numEntities := counterWildcardReadChSize + 10
	p4RtReadClient := &fakeP4RuntimeReadClient{
		recvFn: func() (*p4_v1.ReadResponse, error) {
			if count > 0 {
				return nil, io.EOF
			}
			count += 1
			entities := make([]*p4_v1.Entity, 0, numEntities)
			badEntity := &p4_v1.Entity{
				Entity: &p4_v1.Entity_TableEntry{},
			}
			entities = append(entities, badEntity)
			data := &p4_v1.CounterData{
				ByteCount:   1500,
				PacketCount: 1,
			}
			for i := 1; i < numEntities; i++ {
				entities = append(entities, &p4_v1.Entity{
					Entity: &p4_v1.Entity_CounterEntry{
						CounterEntry: &p4_v1.CounterEntry{
							CounterId: counterId,
							Index:     &p4_v1.Index{Index: int64(i)},
							Data:      data,
						},
					},
				})
			}
			return &p4_v1.ReadResponse{
				Entities: entities,
			}, nil
		},
	}
	p4RtClient := &fakeP4RuntimeClient{
		readFn: func(ctx context.Context, in *p4_v1.ReadRequest, opts ...grpc.CallOption) (p4_v1.P4Runtime_ReadClient, error) {
			return p4RtReadClient, nil
		},
	}
	p4Info := &p4_config_v1.P4Info{
		Counters: []*p4_config_v1.Counter{
			{
				Preamble: &p4_config_v1.Preamble{
					Name: counterName,
					Id:   counterId,
				},
			},
		},
	}
	fakeClient := newTestClient(p4RtClient, p4Info)
	doneCh := make(chan struct{})
	var err error
	go func() {
		defer close(doneCh)
		_, err = fakeClient.ReadCounterEntryWildcard(context.Background(), counterName)
	}()
	timeout := 1 * time.Second
	select {
	case <-doneCh:
		break
	case <-time.After(timeout):
		assert.FailNowf(t, "Timeout", "ReadCounterEntryWildcard should return within %v", timeout)
	}

	assert.Error(t, err, "ReadCounterEntryWildcard should return an error because response includes bad entity")
	assert.EqualError(t, err, "server returned an entity which is not a counter entry!")
}
