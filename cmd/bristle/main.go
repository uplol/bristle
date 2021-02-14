package main

import (
	"fmt"
	"os"

	"github.com/uplol/bristle"
	cli "github.com/urfave/cli/v2"
)

func run(ctx *cli.Context) error {
	server, err := bristle.NewServer(ctx.Path("config"))
	if err != nil {
		return err
	}
	return server.Run()
}

func main() {
	app := &cli.App{
		Name:   "bristle",
		Usage:  "accepts protobuf messages and stores them in clickhouse",
		Action: run,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:  "config",
				Value: "config.json",
				Usage: "path to the configuration file",
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
