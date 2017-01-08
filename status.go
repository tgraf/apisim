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
	statusPort int

	StatusCommand = cli.Command{
		Name:     "status-server",
		Usage:    "HttpServer displaying current status",
		Category: "Function simulation",
		Action:   runStatus,
		Flags: []cli.Flag{
			cli.IntFlag{
				Destination: &statusPort,
				Value:       8888,
				Name:        "p, port",
				Usage:       "Port for status service to listen on",
			},
		},
	}
)

func statusHandler(w http.ResponseWriter, req *http.Request) {
	log.Infof("Status requested %+v", req)

	funcs := make(map[FuncDef]FuncHttp)
	for host, funcPort := range GetExternalFuncTree() {
		for port, funcNode := range funcPort {
			for node := range funcNode {
				uri := fmt.Sprintf("%s:%s%s", host, port, node.path)
				httpFunc, err := NewFuncHttp(node.method, uri)
				if err != nil {
					continue
				}

				funcs[httpFunc] = httpFunc
			}
		}
	}

	result := "[" + FuncMux(funcs, req, FuncHttp{}, NeighborRequest) + "]"
	fmt.Fprintf(w, "jsonCallback(%s);\n", PrettyJSON(result))
}

func runStatus(cli *cli.Context) {
	addr := fmt.Sprintf(":%d", statusPort)
	log.Info("Listening on %s", addr)

	s := manners.NewWithServer(&http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(statusHandler),
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
