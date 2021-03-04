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
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

type BristleClientConfig struct {
	DSN                  url.URL
	TransportCredentials credentials.TransportCredentials
}

type BristleClient struct {
	config *BristleClientConfig
	client v1.BristleIngestService_StreamingClient

	// Locked when we are backing off and no batches should be sent
	backoffUntil uint64
	writerLock   sync.Mutex
	idInc        uint32

	// Batch results
	startupCond *sync.Cond
	results     chan *v1.StreamingServerMessageWriteBatchResult
	outgoing    chan *v1.StreamingClientMessage
}

func NewBristleClient(config *BristleClientConfig) *BristleClient {
	return &BristleClient{
		config:       config,
		backoffUntil: 0,
		idInc:        0,
		startupCond:  sync.NewCond(&sync.Mutex{}),
	}
}

func (b *BristleClient) Run(ctx context.Context) {
	for {
		err := b.runClient(ctx)
		if ctx.Done() != nil && err != nil {
			log.Error().Err(err).Msg("bristle-client: runClient encountered error, retrying in 5 seconds")
			b.startupCond = sync.NewCond(&sync.Mutex{})
			time.Sleep(time.Second * 5)
			continue
		}
		return
	}
}

func (b *BristleClient) runClient(ctx context.Context) error {
	// Create new communication channels and signal we've started up
	b.results = make(chan *v1.StreamingServerMessageWriteBatchResult)
	b.outgoing = make(chan *v1.StreamingClientMessage, 8)
	b.startupCond.Broadcast()
	b.startupCond = nil

	opts := []grpc.DialOption{}

	if b.config.TransportCredentials != nil {
		opts = append(opts, grpc.WithTransportCredentials(b.config.TransportCredentials))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(b.config.DSN.Host, opts...)
	if err != nil {
		return err
	}
	defer func() {
		close(b.results)
		close(b.outgoing)
		conn.Close()
	}()

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

var BatchClientError = errors.New("client encountered internal error writing batch")
var BatchTooBig = errors.New("batch was too big for the upstream to handle")

func (b *BristleClient) WriteBatchSync(messageType string, messages []proto.Message, retryTimes int) error {
	b.writerLock.Lock()
	defer b.writerLock.Unlock()

	// Serialize the messages into a buffer
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

		// Block here if we are waiting for the connection to startup
		if b.startupCond != nil {
			b.startupCond.L.Lock()
			b.startupCond.Wait()
		}

		b.idInc += 1
		b.outgoing <- &v1.StreamingClientMessage{
			Inner: &v1.StreamingClientMessage_WriteBatch{
				WriteBatch: &v1.StreamingClientMessageWriteBatch{
					Id: b.idInc,
					MessageType: &v1.StreamingClientMessageWriteBatch_TypeName{
						TypeName: messageType,
					},
					Length: uint32(len(messages)),
					Data:   data,
				},
			},
		}

		result, ok := <-b.results
		if ok {
			if result.Result == v1.BatchResult_OK {
				// Batch was successful
				return nil
			}

			if result.Result == v1.BatchResult_TOO_BIG {
				// We can't retry this
				return BatchTooBig
			}
		}

		if retryTimes > 0 {
			retryTimes -= 1
			// TODO: more robust backoff calculation here please
			time.Sleep(time.Second * time.Duration(retryTimes))
			continue
		} else if retryTimes == -1 {
			continue
		}

		return fmt.Errorf("WriteBatchSync failed, result was %v", result.Result)
	}
}
