package main

import (
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli"
)

var (
	GenerateK8sSpecCommand = cli.Command{
		Name:     "generate-k8s-spec",
		Usage:    "Generate k8s ReplicationController specs",
		Category: "Kubernetes spec file generation",
		Action:   generateK8sSpec,
	}
	GenerateK8sNetPolicyCommand = cli.Command{
		Name:     "generate-k8s-net-policy",
		Usage:    "Generate k8s NetworkPolicy specs",
		Category: "Kubernetes spec file generation",
		Action:   generateK8sNetPolicy,
	}
)

type PolicyTemplate struct {
	Name   FuncHost
	Policy string
}

func writeSpec(template *template.Template, templateConfig interface{}, path string, typ string) {
	log.Infof("Generating k8s %s %s...", typ, path)
	spec, err := os.Create(path)
	if err != nil {
		log.Fatal("Unable to open spec file \"%s\" for writing: %s", path, err)
	}

	defer spec.Close()

	if err := template.Execute(spec, templateConfig); err != nil {
		log.Fatal("Unable to write spec file: %s", err)
	}
}

func generateK8sNetPolicy(cli *cli.Context) {
	tree := GetExternalFuncTree()

	policyTmpl, err := template.ParseFiles("templates/k8s_net_policy.json")
	if err != nil {
		log.Fatalf("Unable to read template file: %s", err)
	}

	format := "{\n  \"podSelector\": {\n    \"matchLabels\": {\n"
	format += "      \"apisim\": \"%s\"\n    }\n  }\n}"

	for host, funcNode := range tree {
		// status can reach all functions
		policyText := fmt.Sprintf(format, "status")

		for port := range funcNode {
			callers := FindCallers(host, port)
			l4callers := callers.L4Callers()

			for k := range l4callers {
				policyText += "," + fmt.Sprintf(format, k)
			}
		}

		c := PolicyTemplate{host, policyText}
		writeSpec(policyTmpl, c, string(host)+"_netpolicy.spec", "NetPolicy")
	}
}

type TemplateConfig struct {
	Name    FuncHost
	Ports   string
	Command string
}

func generateK8sSpec(cli *cli.Context) {
	tree := GetExternalFuncTree()

	rcTmpl, err := template.ParseFiles("templates/k8s_rc.json")
	if err != nil {
		log.Fatalf("Unable to read template file: %s", err)
	}

	svcTmpl, err := template.ParseFiles("templates/k8s_svc.json")
	if err != nil {
		log.Fatalf("Unable to read template file: %s", err)
	}

	for host, nodeFunc := range tree {
		ports := ""
		nports := 0
		for port := range nodeFunc {
			if nports > 0 {
				ports += ","
			}
			f := "{\"containerPort\": %s, \"name\": \"apisim-%s\"}"
			ports += fmt.Sprintf(f, string(port), string(port))
			nports++
		}

		c := TemplateConfig{host, ports, "\"/go/bin/app\", \"node-server\""}
		writeSpec(rcTmpl, c, string(host)+"_rc.spec", "ReplicationController")

		ports = ""
		nports = 0
		for port := range nodeFunc {
			if nports > 0 {
				ports += ","
			}
			f := "{\"port\": %s, \"targetPort\": \"apisim-%s\"}"
			ports += fmt.Sprintf(f, string(port), string(port))
			nports++
		}

		c = TemplateConfig{host, ports, ""}
		writeSpec(svcTmpl, c, string(host)+"_svc.spec", "Service")
	}
}
