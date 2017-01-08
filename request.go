package main

import (
	"bytes"
	"fmt"
        "time"
	"net/http"
)

type HeaderChangeFunc func(f FuncHttp, inReq *http.Request, outReq *http.Request)

func doRequest(f FuncHttp, inReq *http.Request, readBody bool,
        hdrFunc HeaderChangeFunc, timeout time.Duration) string {
        client := &http.Client{
		Timeout: timeout,
	}

	key := JSON(fmt.Sprintf("%s REQ %s", f.method, f.uri))
	url := fmt.Sprintf("http://%s", f.uri)
	outReq, err := http.NewRequest(f.method, url, nil)
	if err != nil {
                return fmt.Sprintf("{%s: [%s]}", key, ErrorReport(err))
	}

        hdrFunc(f, inReq, outReq)

	resp, err := client.Do(outReq)
	if err != nil {
		return fmt.Sprintf("{%s: [%s]}", key, ErrorReport(err))
	} else if (readBody) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Sprintf("{%s: [%s]}", key, buf.String())
        } else {
                return fmt.Sprintf("\t{%s: %s}", key, JSON("OK"))
	}
}

func pingHeader(f FuncHttp, inReq *http.Request, outReq *http.Request) {
        outReq.Header.Set("NoOperation", "True")
}

func PingRequest(f FuncHttp, inReq *http.Request) string {
        return doRequest(f, inReq, false, pingHeader, Timeout)
}

func requestHeader(f FuncHttp, inReq *http.Request, outReq *http.Request) {
        if inReq.Header.Get("Exploit") != "" {
                outReq.Header.Set("Exploit", "True")
        }

        hdrList, _ := inReq.Header[FuncStackHeader]
        hdrList = append(hdrList, f.String())
        outReq.Header[FuncStackHeader] = hdrList
}

func HttpRequest(f FuncHttp, inReq *http.Request) string {
        return doRequest(f, inReq, true, requestHeader, Timeout * 4)
}

func neighborHeader(f FuncHttp, inReq *http.Request, outReq *http.Request) {
        outReq.Header.Set("NeighborConnectivity", "True")
}

func NeighborRequest(f FuncHttp, inReq *http.Request) string {
        return doRequest(f, inReq, true, neighborHeader, Timeout * 4)
}
