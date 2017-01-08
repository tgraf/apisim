package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/mailgun/manners"
	"github.com/urfave/cli"
)

var (
	hostName     string
	genSpec      bool
	genNetPolicy bool
	genL7Policy  bool
)

var (
	NodeCommand = cli.Command{
		Name:     "node-server",
		Usage:    "Runs a function simulation node",
		Category: "Function simulation",
		Action:   runNode,
		Flags: []cli.Flag{
			cli.StringFlag{
				Destination: &hostName,
				Name:        "n, name",
				Value:       "",
				Usage:       "Name of this function node",
			},
		},
	}
)

func handler(w http.ResponseWriter, req *http.Request) {
	host := hostName
	if host == "" {
		host = req.Host
	}

	if req.Header.Get("NoOperation") != "" {
		return
	}

	uri := host + req.URL.Path
	result := fmt.Sprintf("{%s: [\n", JSON(fmt.Sprintf("%s RESP %s", req.Method, uri)))

	funcName := fmt.Sprintf("%s %s", req.Method, uri)
	def, calls, err := LookupFuncDef(funcName)
	if err != nil {
		result += ErrorReport(err)
	} else if def == nil {
		result += ErrorReport(fmt.Errorf("Function %s not found", funcName))
	} else if req.Header.Get("NeighborConnectivity") != "" {
		log.Infof("Function %+v neighbor connectivity", def)
		result += NeighborConnectivity(req, def)
	} else if req.Header.Get("Exploit") != "" {
		log.Infof("Function %+v being exploited", def)
		exploitCalls := Exploit(req, def)
		result += exploitCalls

		nonHttpCalls := calls.NonHttp()
		if len(nonHttpCalls) > 0 && exploitCalls != "" {
			result += ","
		}
		result += nonHttpCalls.Handle(req)
	} else {
		log.Infof("Function %+v calls: %+v", def, calls)
		result += calls.Handle(req)
	}

	result += "]}"
	fmt.Fprint(w, PrettyJSON(result))

}

func runNode(cli *cli.Context) {
	if hostName != "" {
		hostName = hostName + fmt.Sprintf(":%d", ConfigFuncPort)
	}

	addr := fmt.Sprintf(":%d", ConfigFuncPort)
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
