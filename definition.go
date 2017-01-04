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

type FuncDef interface {
	IsReference() bool
	Handle(w http.ResponseWriter, req *http.Request)
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
		return NewFuncHttp("GET", data), nil
	case "POST":
		return NewFuncHttp("POST", data), nil
	case "PUT":
		return NewFuncHttp("PUT", data), nil
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
