package main

import (
	"fmt"
	"os"
	"text/template"
)

type PolicyTemplate struct {
	Name   string
	Port   string
	Policy string
}

func GenerateNetPolicy() {
	funcs, err := GetUniqueHttpFuncs()
	if err != nil {
		log.Fatalf("%s", err)
	}

	policyTmpl, err := template.ParseFiles("templates/k8s_net_policy.json")
	if err != nil {
		log.Fatalf("Unable to read template file: %s", err)
	}

	format := "{\n  \"podSelector\": {\n    \"matchLabels\": {\n"
	format += "      \"io.cilium.k8s.app\": \"%s\"\n    }\n  }\n}"

	for host, port := range funcs {
		callers := FindCallers(host, port)
		l4callers := callers.L4Callers()

		policyText := ""
		calls := 0

		for k := range l4callers {
			if calls > 0 {
				policyText += ","
			}
			policyText += fmt.Sprintf(format, k)
			calls++
		}

		c := PolicyTemplate{host, port, policyText}

		path := host + "_netpolicy.spec"
		log.Infof("Generating k8s NetPolicy %s...", path)
		policySpec, err := os.Create(path)
		if err != nil {
			log.Fatal("Unable to open spec file \"%s\" for writing: %s", path, err)
		}

		defer policySpec.Close()

		if err := policyTmpl.Execute(policySpec, c); err != nil {
			log.Fatal("Unable to write spec file: %s", err)
		}

	}
}

type TemplateConfig struct {
	Name string
	Port string
}

func GenerateSpecs() {
	funcs, err := GetUniqueHttpFuncs()
	if err != nil {
		log.Fatalf("%s", err)
	}

	rcTmpl, err := template.ParseFiles("templates/k8s_rc.json")
	if err != nil {
		log.Fatalf("Unable to read template file: %s", err)
	}

	svcTmpl, err := template.ParseFiles("templates/k8s_svc.json")
	if err != nil {
		log.Fatalf("Unable to read template file: %s", err)
	}

	for host, _ := range funcs {
		c := TemplateConfig{host, funcs[host]}

		path := host + "_rc.spec"
		log.Infof("Generating k8s ReplicationController %s...", path)
		rcSpec, err := os.Create(path)
		if err != nil {
			log.Fatal("Unable to open spec file \"%s\" for writing: %s", path, err)
		}

		defer rcSpec.Close()

		if err := rcTmpl.Execute(rcSpec, c); err != nil {
			log.Fatal("Unable to write spec file: %s", err)
		}

		path = host + "_svc.spec"
		log.Infof("Generating k8s Service %s...", path)
		svcSpec, err := os.Create(path)
		if err != nil {
			log.Fatal("Unable to open spec file \"%s\" for writing: %s", path, err)
		}

		defer svcSpec.Close()

		if err := svcTmpl.Execute(svcSpec, c); err != nil {
			log.Fatal("Unable to write spec file: %s", err)
		}
	}
}
