package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	v1 "github.com/uplol/bristle/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

type BristleClient struct {
	dsn    *url.URL
	client v1.BristleIngestService_StreamingClient

	// Locked when we are backing off and no batches should be sent
	backoffUntil uint64
	writerLock   sync.Mutex
	idInc        uint32

	// Batch results
	results chan *v1.StreamingServerMessageWriteBatchResult

	outgoing chan *v1.StreamingClientMessage
}

func NewBristleClient(dsn *url.URL) *BristleClient {
	return &BristleClient{
		dsn:          dsn,
		backoffUntil: 0,
		idInc:        0,
		results:      make(chan *v1.StreamingServerMessageWriteBatchResult),
		outgoing:     make(chan *v1.StreamingClientMessage, 8),
	}
}

func (b *BristleClient) Run(ctx context.Context) {
	for {
		err := b.runClient(ctx)
		if ctx.Done() != nil && err != nil {
			log.Error().Err(err).Msg("bristle-client: runClient encountered error, retrying in 5 seconds")
			time.Sleep(time.Second * 5)
			continue
		}
		return
	}
}

func (b *BristleClient) runClient(ctx context.Context) error {
	conn, err := grpc.Dial(b.dsn.Host, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := v1.NewBristleIngestServiceClient(conn)

	stream, err := client.Streaming(ctx)
	if err != nil {
		return err
	}

	go func() {
		for {
			message, ok := <-b.outgoing
			if !ok {
				return
			}

			err := stream.Send(message)
			if err == io.EOF {
				return
			} else if err != nil {
				log.Error().Err(err).Msg("bristle-client: encountered error writing")
				conn.Close()
				return
			}
		}
	}()

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		switch v := in.Inner.(type) {
		case *v1.StreamingServerMessage_WriteBatchResult:
			log.Trace().
				Int("batch", int(v.WriteBatchResult.Id)).
				Int("result", int(v.WriteBatchResult.Result)).
				Msg("bristle-client: batch result received")
			b.results <- v.WriteBatchResult
		case *v1.StreamingServerMessage_Backoff:
			log.Trace().
				Int("until", int(v.Backoff.Until)).
				Interface("types", v.Backoff.Types).
				Msg("bristle-client: backoff request received")
			if v.Backoff.Until > b.backoffUntil {
				atomic.StoreUint64(&b.backoffUntil, v.Backoff.Until)
			}
		}
	}
}

var BatchTooBig = errors.New("batch was too big for the upstream to handle")

func (b *BristleClient) WriteBatchSync(messageType string, messages []proto.Message, retryTimes int) error {
	b.writerLock.Lock()
	defer b.writerLock.Unlock()

	// Serialize the type
	data := []byte{}

	for _, message := range messages {
		messageData, err := proto.Marshal(message)
		if err != nil {
			return err
		}

		data = protowire.AppendBytes(data, messageData)
	}

	for {
		now := uint64(time.Now().UnixNano() / int64(time.Millisecond))
		if b.backoffUntil > now {
			log.Trace().Int("until", int(b.backoffUntil)).Int("now", int(now)).Msg("bristle-client: backing off")
			time.Sleep(time.Millisecond * time.Duration(now-b.backoffUntil))
			continue
		}

		b.idInc += 1
		b.outgoing <- &v1.StreamingClientMessage{
			Inner: &v1.StreamingClientMessage_WriteBatch{
				WriteBatch: &v1.StreamingClientMessageWriteBatch{
					Id:   b.idInc,
					Type: messageType,
					Size: uint32(len(messages)),
					Data: data,
				},
			},
		}

		result := <-b.results
		if result.Result == v1.BatchResult_OK {
			return nil
		}

		// We can't retry this
		if result.Result == v1.BatchResult_TOO_BIG {
			return BatchTooBig
		}

		if retryTimes > 0 {
			retryTimes -= 1
			continue
		} else if retryTimes == -1 {
			continue
		}

		return fmt.Errorf("WriteBatchSync failed, result was %v", result.Result)
	}
}
