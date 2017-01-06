package main

import (
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli"
)

var (
	L7PolicyGenerateCommand = cli.Command{
		Name:     "generate-l7-policy",
		Usage:    "Generate L7 policy",
		Category: "L7 policy generation",
		Action:   generateL7Policy,
	}
)

type L7Template struct {
	Name   FuncHost
	Policy string
}

func writeL7Policy(host FuncHost, port FuncPort, node ExternalFuncNode) {
	policyTmpl, err := template.ParseFiles("templates/l7_policy.json")
	if err != nil {
		log.Fatalf("Unable to read template file: %s", err)
	}

	policyText := ""
	ncalls := 0
	for _, calls := range node {
		for _, call := range calls {
			switch call.(type) {
			case FuncHttp:
				hf := call.(FuncHttp)
				if ncalls > 0 {
					policyText += ",\n"
				}
				policyText += fmt.Sprintf("\t\t{%s %s}", hf.method, hf.uri)
				ncalls++
			}
		}
	}

	tmpl := L7Template{host, policyText}
	path := string(host) + "_" + string(port) + "_l7policy.spec"
	log.Infof("Generating L7 Policy %s...", path)
	policySpec, err := os.Create(path)
	if err != nil {
		log.Fatal("Unable to open spec file \"%s\" for writing: %s", path, err)
	}

	defer policySpec.Close()

	if err := policyTmpl.Execute(policySpec, tmpl); err != nil {
		log.Fatal("Unable to write spec file: %s", err)
	}
}

func generateL7Policy(cli *cli.Context) {
	tree := GetExternalFuncTree()

	for host, funcPort := range tree {
		for port, funcNode := range funcPort {
			writeL7Policy(host, port, funcNode)
		}
	}
}
