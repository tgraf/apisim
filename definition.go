package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

var (
	definitionTree = NewFuncTree()
)

type FuncHost string
type FuncPort string
type FuncNode string // method + path, e.g. "GET /""
type FuncDef interface {
	IsReference() bool
	Handle(req *http.Request) string
	String() string
}

type FuncCalls []FuncDef

type FuncTree struct {
	Funcs map[FuncDef]FuncCalls
}

func NewFuncTree() *FuncTree {
	return &FuncTree{
		Funcs: make(map[FuncDef]FuncCalls),
	}
}

func ParseFuncType(name string, data string) (FuncDef, error) {
	switch name {
	case "GET":
		return NewFuncHttp("GET", data)
	case "POST":
		return NewFuncHttp("POST", data)
	case "PUT":
		return NewFuncHttp("PUT", data)
	case "CALL":
		return NewFuncCall(data), nil
	case "DATA":
		return NewFuncData(data), nil
	default:
		return nil, fmt.Errorf("unknown type \"%s\"", name)
	}
}

func LookupFuncDef(name string) (FuncDef, FuncCalls, error) {
	def, err := ParseFuncDef(name)
	if err != nil {
		return nil, nil, err
	}

	if calls, ok := definitionTree.Funcs[def]; ok {
		return def, calls, nil
	}

	return nil, nil, nil
}

type HttpCallers []FuncHttp

func (c HttpCallers) L4Callers() map[FuncHost]FuncPort {
	result := make(map[FuncHost]FuncPort)
	for _, caller := range c {
		result[caller.host] = caller.port
	}
	return result
}

func FindCallers(host FuncHost, port FuncPort) HttpCallers {
	result := []FuncHttp{}

	for key, _ := range definitionTree.Funcs {

		switch key.(type) {
		case FuncHttp:
			httpFunc := key.(FuncHttp)

			for _, call := range definitionTree.Funcs[key] {
				switch call.(type) {
				case FuncHttp:
					httpCall := call.(FuncHttp)

					if httpCall.host == host &&
						httpCall.port == port {
						result = append(result, httpFunc)
					}
				}
			}
		}
	}

	return result
}

type ExternalFuncNode map[FuncNode]FuncCalls
type ExternalFuncPort map[FuncPort]ExternalFuncNode
type ExternalFuncTree map[FuncHost]ExternalFuncPort

func GetExternalFuncTree() ExternalFuncTree {
	result := make(ExternalFuncTree)

	for key, calls := range definitionTree.Funcs {
		switch key.(type) {
		case FuncHttp:
			hf := key.(FuncHttp)

			if _, ok := result[hf.host]; !ok {
				result[hf.host] = make(ExternalFuncPort)
			}

			if _, ok := result[hf.host][hf.port]; !ok {
				result[hf.host][hf.port] = make(ExternalFuncNode)
			}

			node := FuncNode(hf.method + " " + hf.path)
			result[hf.host][hf.port][node] = calls
		}
	}

	return result
}

type HttpCalls map[FuncHost]map[string]FuncHttp

func GetUniqueHttpCalls() HttpCalls {
	result := make(HttpCalls)

	for key := range definitionTree.Funcs {
		switch key.(type) {
		case FuncHttp:
			hf := key.(FuncHttp)

			if _, ok := result[hf.host]; !ok {
				result[hf.host] = make(map[string]FuncHttp)
			}

			for _, call := range definitionTree.Funcs[key] {
				switch call.(type) {
				case FuncHttp:
					c := call.(FuncHttp)
					cKey := fmt.Sprintf("%s %s", c.method, c.uri)
					result[hf.host][cKey] = c
				}
			}
		}
	}

	return result
}

func (c FuncCalls) NonHttp() FuncCalls {
	res := make(FuncCalls, 0)
	for k := range c {
		switch c[k].(type) {
		case FuncHttp:
			continue
		}

		res = append(res, c[k])
	}

	return res
}

func (c FuncCalls) Http() map[FuncDef]FuncHttp {
	res := make(map[FuncDef]FuncHttp)
	for k := range c {
		key := c[k]
		switch key.(type) {
		case FuncHttp:
			res[key] = key.(FuncHttp)
		}
	}

	return res
}

func GetHttpFuncs(req *http.Request) map[FuncDef]FuncHttp {
	result := make(map[FuncDef]FuncHttp)

	for key, _ := range definitionTree.Funcs {
		switch key.(type) {
		case FuncHttp:
			// If req is provided, ignored funcs in the stack
			if req != nil && FuncInHeader(req, key.String()) {
				log.Infof("Ignoring recursive call %v", key)
				continue
			}
			result[key] = key.(FuncHttp)
		}
	}

	return result
}

func ParseFuncDef(key string) (FuncDef, error) {
	var t, data string
	if n, err := fmt.Sscanf(key, "%s %s", &t, &data); n != 2 || err != nil {
		return nil, fmt.Errorf("invalid key \"%s\": %s", key, err.Error())
	}

	ft, err := ParseFuncType(t, data)
	if err != nil {
		return nil, err
	}

	return ft, nil
}

type FuncCallsJSON []string

func (f *FuncTree) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty function tree")
	}

	var pt struct {
		Funcs map[string]FuncCallsJSON `json:"Functions"`
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&pt)
	if err != nil {
		return fmt.Errorf("invalid json: %s", err)
	}

	for key, _ := range pt.Funcs {
		def, err := ParseFuncDef(key)
		if err != nil {
			return err
		}

		calls := pt.Funcs[key]
		f.Funcs[def] = make(FuncCalls, len(calls))
		for i, call := range calls {
			callDef, err := ParseFuncDef(call)
			if err != nil {
				return err
			}

			if callDef.IsReference() {
				if _, ok := pt.Funcs[call]; !ok {
					return fmt.Errorf("unable to find key \"%v\"", call)
				}
			}

			f.Funcs[def][i] = callDef
		}
	}

	return nil
}

func getContext(content []byte, offset int64) (int, string, int) {
	if offset >= int64(len(content)) || offset < 0 {
		return 0, fmt.Sprintf("[error: Offset %d is out of bounds 0..%d]", offset, len(content)), 0
	}

	lineN := strings.Count(string(content[:offset]), "\n") + 1

	start := strings.LastIndexByte(string(content[:offset]), '\n')
	if start == -1 {
		start = 0
	} else {
		start++
	}

	end := strings.IndexByte(string(content[start:]), '\n')
	l := ""
	if end == -1 {
		l = string(content[start:])
	} else {
		end = end + start
		l = string(content[start:end])
	}

	return lineN, l, (int(offset) - start)
}

func handleUnmarshalError(f string, content []byte, err error) error {
	switch e := err.(type) {
	case *json.SyntaxError:
		line, ctx, off := getContext(content, e.Offset)

		if off <= 1 {
			return fmt.Errorf("empty json definition")
		}

		preoff := off - 1
		pre := make([]byte, preoff)
		copy(pre, ctx[:preoff])
		for i := 0; i < preoff && i < len(pre); i++ {
			if pre[i] != '\t' {
				pre[i] = ' '
			}
		}

		return fmt.Errorf("Error: %s:%d: syntax error at offset %d:\n%s\n%s^",
			path.Base(f), line, off, ctx, pre)
	case *json.UnmarshalTypeError:
		line, ctx, off := getContext(content, e.Offset)
		return fmt.Errorf("Error: %s:%d: unable to assign value '%s' to type '%v':\n%s\n%*c",
			path.Base(f), line, e.Value, e.Type, ctx, off, '^')
	default:
		return fmt.Errorf("Error: %s: unknown error: %s", path.Base(f), err)
	}
}

func ReadConfig(path string) error {
	log.Infof("Loading configuration file %s", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(content, definitionTree); err != nil {
		return handleUnmarshalError(path, content, err)
	}

	return nil
}
