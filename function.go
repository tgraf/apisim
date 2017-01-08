package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	FuncStackHeader = http.CanonicalHeaderKey("FuncStack")
)

func JSON(text string) string {
	s, _ := json.Marshal(text)
	return string(s)
}

func ErrorReport(err error) string {
	return JSON("ERROR: " + err.Error())
}

type FuncData struct {
	data string
}

func NewFuncData(data string) FuncData {
	return FuncData{data: data}
}

func (f FuncData) IsReference() bool { return false }
func (f FuncData) String() string    { return "DATA " + f.data }
func (f FuncData) Handle(req *http.Request) string {
	return fmt.Sprintf("{\"DATA\": %s}", JSON(f.data))
}

type FuncCall struct {
	name string
}

func NewFuncCall(name string) FuncCall {
	return FuncCall{name: name}
}

func (c FuncCalls) Handle(req *http.Request) string {
	reply := FuncMux(c.Http(), req, FuncHttp{}, HttpRequest)

	for k := range c.NonHttp() {
		if reply != "" {
			reply += ","
		}
		reply += c[k].Handle(req)
	}

	return reply
}

func (f FuncCall) IsReference() bool { return true }
func (f FuncCall) String() string    { return "CALL " + f.name }
func (f FuncCall) Handle(req *http.Request) string {
	key := JSON(fmt.Sprintf("CALL %s", f.name))

	calls, ok := definitionTree.Funcs[f]
	if !ok {
		return fmt.Sprintf("{%s: [\"Function not found\"]}", key)
	}

	return fmt.Sprintf("{%s: [%s]}", key, calls.Handle(req))
}

type FuncHttp struct {
	method string
	uri    string
	host   FuncHost
	port   FuncPort
	path   string
}

func NewFuncHttp(method string, uri string) (FuncHttp, error) {
	url, err := url.Parse("http://" + uri)
	if err != nil {
		return FuncHttp{}, err
	}

	if !strings.Contains(url.Host, ":") {
		url.Host = url.Host + fmt.Sprintf(":%d", ConfigFuncPort)
		uri = url.Host + url.Path
	}

	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		return FuncHttp{}, fmt.Errorf("Unable derive host and port from \"%s\"", url.Host)
	}

	return FuncHttp{
		method: method,
		uri:    uri,
		host:   FuncHost(host),
		port:   FuncPort(port),
		path:   url.Path,
	}, nil
}

func (f FuncHttp) IsReference() bool { return true }
func (f FuncHttp) String() string    { return fmt.Sprintf("%s %s", f.method, f.uri) }
func (f FuncHttp) Handle(req *http.Request) string {
	return HttpRequest(f, req)
}

func FuncInHeader(req *http.Request, name string) bool {
	if hdrList, ok := req.Header[FuncStackHeader]; ok {
		for _, v := range hdrList {
			if v == name {
				return true
			}
		}
	}

	return false
}

type RequestFunc func(http FuncHttp, inReq *http.Request) string

func FuncMux(funcs map[FuncDef]FuncHttp, inReq *http.Request, ownFunc FuncDef, reqFunc RequestFunc) string {
	responses := make(chan string, 32)
	var wg sync.WaitGroup

	nreplies := len(funcs)
	if nreplies == 0 {
		return ""
	}
	log.Infof("Waiting for %d responses", nreplies)
	wg.Add(nreplies)

	for key := range funcs {
		go func(key FuncDef) {
			defer wg.Done()
			log.Infof("Scheduling %+v", key)
			if key.String() == ownFunc.String() {
				responses <- fmt.Sprintf("\t{%s: %s}", JSON(key.String()), JSON("NOP"))
			} else {
				responses <- "\t" + reqFunc(funcs[key], inReq)
			}
			log.Infof("Done with %+v", key)
		}(key)
	}

	wg.Wait()
	log.Infof("Reading responses")

	ret := ""
	for i := 0; i < nreplies; i++ {
		str := <-responses
		if i > 0 {
			ret += ",\n"
		}
		ret += str
	}

	log.Infof("Read all responses")

	return ret
}

func Exploit(req *http.Request, ownFunc FuncDef) string {
	return FuncMux(GetHttpFuncs(req), req, ownFunc, HttpRequest)
}

func NeighborConnectivity(req *http.Request, ownFunc FuncDef) string {
	return FuncMux(GetHttpFuncs(req), req, ownFunc, PingRequest)
}
