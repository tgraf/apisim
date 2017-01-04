package main

import (
	"os"
	"text/template"
)

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
