package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
}

func NewFuncHttp(method string, uri string) FuncHttp {
	if !strings.Contains(uri, ":") {
		uri = "http://" + uri
		if u, err := url.Parse(uri); err == nil {
			u.Host = u.Host + fmt.Sprintf(":%d", ListenPort)
			uri = u.Host + u.Path
		}
	}

	return FuncHttp{
		method: method,
		uri:    uri,
	}
}

func (f FuncHttp) IsReference() bool { return true }
func (f FuncHttp) Handle(w http.ResponseWriter, req *http.Request) {
	client := &http.Client{}

	key := JSON(fmt.Sprintf("%s REQ %s", f.method, f.uri))
	url := fmt.Sprintf("http://%s", f.uri)
	outReq, err := http.NewRequest(f.method, url, nil)
	if err != nil {
		fmt.Fprintf(w, "{%s: [%s]}", key, JSON(err.Error()))
		return
	}

	resp, err := client.Do(outReq)
	if err != nil {
		fmt.Fprintf(w, "{%s: [%s]}", key, JSON(err.Error()))
		return
	}

	fmt.Fprintf(w, "{%s: [", key)
	io.Copy(w, resp.Body)
	fmt.Fprintf(w, "]}")
}
