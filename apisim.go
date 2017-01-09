package main

import (
	"os"
	"time"

	"github.com/op/go-logging"
	"github.com/urfave/cli"
)

var (
	log            = logging.MustGetLogger("apisim")
	configFile     string
	ConfigFuncPort int
	CliCommand     cli.Command
	Timeout        = 20 * time.Second
)

func main() {
	app := cli.NewApp()
	app.Name = "apisim"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Destination: &configFile,
			Name:        "c, config",
			Value:       "definition.json",
			Usage:       "Path to configuration file",
		},
		cli.IntFlag{
			Destination: &ConfigFuncPort,
			Name:        "func-port",
			Value:       8080,
			Usage:       "Port for functions to listen on",
		},
	}
	app.Commands = []cli.Command{
		NodeCommand,
		StatusCommand,
		GenerateK8sSpecCommand,
		GenerateK8sNetPolicyCommand,
		L7PolicyGenerateCommand,
	}
	app.Before = initEnv

	app.Run(os.Args)
}

func initEnv(ctx *cli.Context) error {
	if err := ReadConfig(configFile); err != nil {
		log.Fatal(err)
	}

	return nil
}
