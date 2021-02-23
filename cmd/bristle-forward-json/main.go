package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uplol/bristle"
	"github.com/uplol/bristle/client"
	v1 "github.com/uplol/bristle/proto/v1"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func stdinProcessor(messageType protoreflect.MessageType, batcher *client.BristleBatcher) error {
	body := messageType.New().Interface()
	typeName := string(messageType.Descriptor().FullName())
	reader := bufio.NewReader(os.Stdin)
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}

		err = protojson.Unmarshal(data, body)
		if err != nil {
			log.Printf("%v", string(data))
			return err
		}

		err = batcher.WriteBatch(typeName, []proto.Message{body})
		if err != nil {
			log.Printf("OOF")
			return err
		}
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
	var err error

	for {
		payload := <-payloads

		for {
			for conn == nil {
				conn, err = getClientConn(destination)
				if err != nil {
					log.Warn().Err(err).Msg("sender: backing off due to connection error")
					time.Sleep(time.Second)
					continue
				}
				client = v1.NewBristleIngestServiceClient(conn)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			_, err := client.WriteBatch(ctx, &v1.WriteBatchRequest{
				Payloads: []*v1.Payload{
					payload,
				},
			})
			cancel()

			if err != nil {
				log.Warn().Err(err).Msg("sender: backing off due to send error")
				conn.Close()
				conn = nil
				time.Sleep(time.Second)
				continue
			}

			break
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

	done := make(chan struct{})
	payloads := make(chan *v1.Payload)

	upstreamURL, err := url.Parse(ctx.String("upstream"))
	if err != nil {
		return err
	}

	bristleClient := client.NewBristleClient(upstreamURL)
	bristleBatcher := client.NewBristleBatcher(bristleClient, client.BristleBatcherConfig{
		BufferSize:    100000,
		Retry:         true,
		FlushInterval: time.Second * 5,
	})

	bgCtx, cancel := context.WithCancel(context.Background())
	go bristleClient.Run(bgCtx)
	go bristleBatcher.Run(bgCtx)

	go func() {
		err := stdinProcessor(messageType, bristleBatcher)
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
				Value: "bristle://localhost:8122",
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
