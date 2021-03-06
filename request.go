package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

type HeaderChangeFunc func(f FuncHttp, inReq *http.Request, outReq *http.Request)

func doRequest(ownFunc FuncDef, f FuncHttp, inReq *http.Request, readBody bool,
	hdrFunc HeaderChangeFunc, timeout time.Duration) string {
	client := &http.Client{
		Timeout: timeout,
	}

	key := JSON(fmt.Sprintf("%s %s", f.method, f.uri))
	url := fmt.Sprintf("http://%s", f.uri)
	outReq, err := http.NewRequest(f.method, url, nil)
	if err != nil {
		return fmt.Sprintf("{%s: %s}", ErrorReport(err))
	}

	hdrFunc(f, inReq, outReq)

	resp, err := client.Do(outReq)
	if err != nil {
		return fmt.Sprintf("{%s: %s}", key, ErrorReport(err))
	} else if readBody {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Sprintf("{%s: %s}", key, buf.String())
	} else {
		if IsCaller(ownFunc, f) {
			return fmt.Sprintf("{%s: %s}", key, JSON("OK"))
		} else {
			return fmt.Sprintf("{%s: %s}", key, JSON("VULN"))
		}
	}
}

func pingHeader(f FuncHttp, inReq *http.Request, outReq *http.Request) {
	outReq.Header.Set("NoOperation", "True")
}

func PingRequest(ownFunc FuncDef, f FuncHttp, inReq *http.Request) string {
	return doRequest(ownFunc, f, inReq, false, pingHeader, Timeout)
}

func requestHeader(f FuncHttp, inReq *http.Request, outReq *http.Request) {
	if inReq.Header.Get("Exploit") != "" {
		outReq.Header.Set("Exploit", "True")
	}

	hdrList, _ := inReq.Header[FuncStackHeader]
	hdrList = append(hdrList, f.String())
	outReq.Header[FuncStackHeader] = hdrList
}

func HttpRequest(ownFunc FuncDef, f FuncHttp, inReq *http.Request) string {
	return doRequest(ownFunc, f, inReq, true, requestHeader, Timeout*4)
}

func neighborHeader(f FuncHttp, inReq *http.Request, outReq *http.Request) {
	outReq.Header.Set("NeighborConnectivity", "True")
}

func NeighborRequest(ownFunc FuncDef, f FuncHttp, inReq *http.Request) string {
	return doRequest(ownFunc, f, inReq, true, neighborHeader, Timeout*4)
}
