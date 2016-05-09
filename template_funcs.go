package main

import (
	"os"
	"text/template"
)

func getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"hostname": func() string {
			hostname, _ := os.Hostname()
			return hostname
		},
		"env": func(key string) string {
			value, _ := os.LookupEnv(key)
			return value
		},
	}
}
