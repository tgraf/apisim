package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/mailgun/manners"
	"github.com/op/go-logging"
	"github.com/urfave/cli"
)

var (
	ListenPort   int
	configFile   string
	hostName     string
	genSpec      bool
	genNetPolicy bool
	log          = logging.MustGetLogger("apisim")
	CliCommand   cli.Command
)

func main() {
	app := cli.NewApp()
	app.Name = "apisim"
	app.Usage = "Simulates API calls in a mesh of functions"
	app.Action = run
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Destination: &ListenPort,
			Name:        "p, port",
			Value:       8080,
			Usage:       "Port to listen on",
		},
		cli.StringFlag{
			Destination: &configFile,
			Name:        "c, config",
			Value:       "definition.json",
			Usage:       "Path to configuration file",
		},
		cli.StringFlag{
			Destination: &hostName,
			Name:        "n, name",
			Value:       "",
			Usage:       "Name of this function node",
		},
		cli.BoolFlag{
			Destination: &genSpec,
			Name:        "generate-k8s-spec",
			Usage:       "Generate k8s ReplicationController specs",
		},
		cli.BoolFlag{
			Destination: &genNetPolicy,
			Name:        "generate-k8s-net-policy",
			Usage:       "Generate k8s NetworkPolicy specs",
		},
	}
	app.Run(os.Args)
}

func handler(w http.ResponseWriter, req *http.Request) {
	host := hostName
	if host == "" {
		host = req.Host
	}

	if req.Header.Get("NoOperation") != "" {
		return
	}

	uri := host + req.URL.Path
	fmt.Fprintf(w, "{%s: [", JSON(fmt.Sprintf("%s RESP %s", req.Method, uri)))

	funcName := fmt.Sprintf("%s %s", req.Method, uri)
	def, calls, err := LookupFuncDef(funcName)
	if err != nil {
		fmt.Fprintf(w, "%s", JSON(err.Error()))
	} else if def == nil {
		fmt.Fprintf(w, "%s", JSON(fmt.Sprintf("Function %s not found", funcName)))
	} else if req.Header.Get("NeighbourConnectivity") != "" {
		log.Infof("Function %+v neighbor connectivity")
		NeighborConnectivity(w, req, def)
	} else if req.Header.Get("Exploit") != "" {
		log.Infof("Function %+v being exploited", def)
		exploitCalls := Exploit(w, req, def)

		nonHttpCalls := calls.NonHttp()
		if len(nonHttpCalls) > 0 && exploitCalls > 0 {
			fmt.Fprintf(w, ",")
		}
		nonHttpCalls.Handle(w, req)
	} else {
		log.Infof("Function %+v calls: %+v", def, calls)
		calls.Handle(w, req)
	}

	fmt.Fprintf(w, "]}")
}

func run(cli *cli.Context) {
	if hostName != "" {
		hostName = hostName + fmt.Sprintf(":%d", ListenPort)
	}

	if err := ReadConfig(configFile); err != nil {
		log.Fatal(err)
	}

	if genSpec {
		GenerateSpecs()
		return
	}

	if genNetPolicy {
		GenerateNetPolicy()
		return
	}

	addr := fmt.Sprintf(":%d", ListenPort)
	log.Info("Listening on %s", addr)

	s := manners.NewWithServer(&http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(handler),
	})

	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt, os.Kill)
		<-sigchan
		log.Info("Shutting down...")
		s.Close()
	}()

	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
