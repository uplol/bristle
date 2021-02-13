package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uplol/bristle"
	v1 "github.com/uplol/bristle/proto/v1"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Batcher struct {
	sync.Mutex

	typeName      string
	flushInterval time.Duration
	buffer        [][]byte
}

func NewBatcher(typeName string, flushInterval time.Duration) *Batcher {
	return &Batcher{
		typeName:      typeName,
		flushInterval: flushInterval,
		buffer:        make([][]byte, 0),
	}
}

func (b *Batcher) Add(data []byte) {
	b.Lock()
	defer b.Unlock()
	b.buffer = append(b.buffer, data)
}

func (b *Batcher) flush() *v1.Payload {
	b.Lock()
	defer b.Unlock()
	if len(b.buffer) == 0 {
		return nil
	}

	payload := &v1.Payload{
		Type: b.typeName,
		Body: b.buffer,
	}
	b.buffer = make([][]byte, 0)
	return payload
}

func (b *Batcher) RunFlusher(ctx context.Context, destination chan<- *v1.Payload, done chan struct{}) {
	ticker := time.NewTicker(b.flushInterval)
	defer close(done)

	for {
		select {
		case <-ticker.C:
			batch := b.flush()
			if batch != nil {
				destination <- batch
				log.Debug().Int("batch-size", len(batch.Body)).Msg("flusher: batch flushed to sender")
				continue
			}
		}

		select {
		case <-ctx.Done():
			log.Printf("flusher is done!")
			return
		default:
			continue
		}
	}
}

func stdinProcessor(messageType protoreflect.MessageType, batcher *Batcher) error {
	body := messageType.New().Interface()
	reader := bufio.NewReader(os.Stdin)
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}

		err = protojson.Unmarshal(data, body)
		if err != nil {
			return err
		}

		binaryData, err := proto.Marshal(body)
		if err != nil {
			return err
		}
		batcher.Add(binaryData)
	}
}

func getClientConn(destination string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(destination, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func upstreamForwarder(typeName, destination string, payloads <-chan *v1.Payload) error {
	var conn *grpc.ClientConn
	var client v1.BristleIngestServiceClient
	var stream v1.BristleIngestService_StreamingWriteBatchClient
	var err error

	for {
		if conn == nil {
			for {
				conn, err = getClientConn(destination)
				if err != nil {
					log.Warn().Err(err).Msg("sender: backing off due to connection error")
					time.Sleep(time.Second)
					continue
				}
				client = v1.NewBristleIngestServiceClient(conn)

				stream, err = client.StreamingWriteBatch(context.Background())
				if err != nil {
					log.Warn().Err(err).Msg("sender: backing off due to streaming error")
					time.Sleep(time.Second)
					continue
				}
				break
			}
		}

		for {
			select {
			case payload := <-payloads:
				err := stream.Send(&v1.WriteBatchRequest{
					Payloads: []*v1.Payload{
						payload,
					},
				})
				if err != nil {
					break
				}
				continue
			}
		}
	}
}

func run(ctx *cli.Context) error {
	logLevel := ctx.String("log-level")

	switch logLevel {
	case "error":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		return fmt.Errorf("unknown logging level '%v'", logLevel)
	}

	registry := bristle.NewProtoRegistry()
	for _, path := range ctx.StringSlice("register-proto") {
		err := registry.RegisterPath(path)
		if err != nil {
			return err
		}
	}

	selectedMessageTypeName := ctx.String("type")
	if selectedMessageTypeName == "" {
		return errors.New("must provide a 'type'")
	}

	messageType, ok := registry.MessageTypes[protoreflect.FullName(selectedMessageTypeName)]
	if !ok {
		return errors.New("could not find registered message for type: " + selectedMessageTypeName)
	}

	bgCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	payloads := make(chan *v1.Payload)

	batcher := NewBatcher(selectedMessageTypeName, time.Duration(ctx.Int("flush-interval"))*time.Millisecond)

	go batcher.RunFlusher(bgCtx, payloads, done)

	go func() {
		err := stdinProcessor(messageType, batcher)
		if err != nil {
			log.Error().Err(err).Msg("stdin: encountered error")
		}
		cancel()
	}()

	go func() {
		err := upstreamForwarder(selectedMessageTypeName, ctx.String("upstream"), payloads)
		if err != nil {
			log.Error().Err(err).Msg("sender: encountered error")
		}
		cancel()
	}()

	<-done
	close(payloads)
	return nil
}

func main() {
	app := &cli.App{
		Name:   "bristle-forward-json",
		Usage:  "forward json payloads of configured protos to a central bristle ingest service",
		Action: run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "upstream",
				Value: "localhost:8122",
				Usage: "address of the upstream bristle ingest service we will forward payloads too",
			},
			&cli.StringFlag{
				Name:  "type",
				Usage: "the fully qualified protobuf message type name",
			},
			&cli.StringSliceFlag{
				Name:  "register-proto",
				Usage: "register a individual or directory of binary proto descriptors",
			},
			&cli.IntFlag{
				Name:  "flush-interval",
				Usage: "the interval (in milliseconds) that we will flush at",
				Value: 1000,
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "set the logging level",
				Value: "info",
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
