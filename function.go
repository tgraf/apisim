package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var (
	FuncStackHeader = http.CanonicalHeaderKey("FuncStack")
)

func JSON(text string) string {
	s, _ := json.Marshal(text)
	return string(s)
}

type FuncData struct {
	data string
}

func NewFuncData(data string) FuncData {
	return FuncData{data: data}
}

func (f FuncData) IsReference() bool { return false }
func (f FuncData) String() string    { return "DATA " + f.data }
func (f FuncData) Handle(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "{\"DATA\": %s}", JSON(f.data))
}

type FuncCall struct {
	name string
}

func NewFuncCall(name string) FuncCall {
	return FuncCall{name: name}
}

func (c FuncCalls) Handle(w http.ResponseWriter, req *http.Request) {
	for k := range c {
		c[k].Handle(w, req)
		if k != len(c)-1 {
			fmt.Fprintf(w, ",")
		}
	}
}

func (f FuncCall) IsReference() bool { return true }
func (f FuncCall) String() string    { return "CALL " + f.name }
func (f FuncCall) Handle(w http.ResponseWriter, req *http.Request) {
	key := JSON(fmt.Sprintf("CALL %s", f.name))

	calls, ok := definitionTree.Funcs[f]
	if !ok {
		fmt.Fprintf(w, "{%s: [\"Function not found\"]}", key)
		return
	}

	fmt.Fprintf(w, "{%s: [", key)
	calls.Handle(w, req)
	fmt.Fprintf(w, "]}")
}

type FuncHttp struct {
	method string
	uri    string
	host   string
	port   string
}

func NewFuncHttp(method string, uri string) (FuncHttp, error) {
	url, err := url.Parse("http://" + uri)
	if err != nil {
		return FuncHttp{}, err
	}

	if !strings.Contains(url.Host, ":") {
		url.Host = url.Host + fmt.Sprintf(":%d", ListenPort)
		uri = url.Host + url.Path
	}

	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		return FuncHttp{}, fmt.Errorf("Unable derive host and port from \"%s\"", url.Host)
	}

	return FuncHttp{
		method: method,
		uri:    uri,
		host:   host,
		port:   port,
	}, nil
}

func (f FuncHttp) IsReference() bool { return true }
func (f FuncHttp) String() string    { return fmt.Sprintf("%s %s", f.method, f.uri) }
func (f FuncHttp) Handle(w http.ResponseWriter, req *http.Request) {
	client := &http.Client{}

	key := JSON(fmt.Sprintf("%s REQ %s", f.method, f.uri))
	url := fmt.Sprintf("http://%s", f.uri)
	outReq, err := http.NewRequest(f.method, url, nil)
	if err != nil {
		fmt.Fprintf(w, "{%s: [%s]}", key, JSON(err.Error()))
		return
	}

	if req.Header.Get("Exploit") != "" {
		outReq.Header.Set("Exploit", "True")
	}

	hdrList, _ := req.Header[FuncStackHeader]
	hdrList = append(hdrList, f.String())
	outReq.Header[FuncStackHeader] = hdrList

	resp, err := client.Do(outReq)
	if err != nil {
		fmt.Fprintf(w, "{%s: [%s]}", key, JSON(err.Error()))
		return
	}

	fmt.Fprintf(w, "{%s: [", key)
	io.Copy(w, resp.Body)
	fmt.Fprintf(w, "]}")
}

func funcInHeader(req *http.Request, name string) bool {
	if hdrList, ok := req.Header[FuncStackHeader]; ok {
		for _, v := range hdrList {
			if v == name {
				return true
			}
		}
	}

	return false
}

func Exploit(w http.ResponseWriter, req *http.Request, ownFunc FuncDef) int {
	funcs := GetHttpFuncs()
	calls := 0

	for k, _ := range funcs {
		if k.String() == ownFunc.String() {
			log.Infof("Ignoring function \"%s\" (own)", k.String())
			continue
		}
		if funcInHeader(req, k.String()) {
			log.Infof("Ignoring function \"%s\" (found in stack)", k.String())
			continue
		}

		if calls > 0 {
			fmt.Fprintf(w, ",")
		}

		calls++
		funcs[k].Handle(w, req)
	}

	return calls
}

func (f FuncHttp) Ping(w http.ResponseWriter, req *http.Request) {
	client := &http.Client{}

	key := JSON(fmt.Sprintf("%s %s", f.method, f.uri))
	url := fmt.Sprintf("http://%s", f.uri)
	outReq, err := http.NewRequest(f.method, url, nil)
	if err != nil {
		fmt.Fprintf(w, "{%s: [%s]}", key, JSON(err.Error()))
		return
	}

	outReq.Header.Set("NoOperation", "True")

	_, err = client.Do(outReq)
	if err != nil {
		fmt.Fprintf(w, "{%s: %s}", key, JSON(err.Error()))
	} else {
		fmt.Fprintf(w, "{%s: %s}", key, JSON("OK"))
	}
}

func NeighborConnectivity(w http.ResponseWriter, req *http.Request, ownFunc FuncDef) int {
	funcs := GetHttpFuncs()
	calls := 0

	for k, _ := range funcs {
		if k.String() == ownFunc.String() {
			log.Infof("Ignoring function \"%s\" (own)", k.String())
			continue
		}
		if funcInHeader(req, k.String()) {
			log.Infof("Ignoring function \"%s\" (found in stack)", k.String())
			continue
		}

		if calls > 0 {
			fmt.Fprintf(w, ",")
		}

		calls++
		funcs[k].Ping(w, req)
	}

	return calls
}
