package main

import (
	"fmt"
	"io"
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

	tree := GetExternalFuncTree()

	calls := 0
	for host, funcNode := range tree {
		for port := range funcNode {
			if calls > 0 {
				fmt.Fprintf(w, ",\n")
			}

			calls++
			client := &http.Client{
				Timeout: Timeout,
			}

			key := JSON(fmt.Sprintf("%s:%s", host, port))
			url := fmt.Sprintf("http://%s:%s/", host, port)
			outReq, err := http.NewRequest("GET", url, nil)
			if err != nil {
				fmt.Fprintf(w, "{%s: [%s]}", key, JSON(err.Error()))
				return
			}

			outReq.Header.Set("NeighborConnectivity", "True")

			resp, err := client.Do(outReq)
			if err != nil {
				fmt.Fprintf(w, "{%s: [%s]}", key, JSON(err.Error()))
				return
			}

			fmt.Fprintf(w, "{%s: [\n", key)
			io.Copy(w, resp.Body)
			fmt.Fprintf(w, "]}")
		}
	}
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
