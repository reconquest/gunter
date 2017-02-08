package main

import (
	"os"
	"path/filepath"
	"plugin"
	"strings"
	libtemplate "text/template"

	"github.com/reconquest/ser-go"
)

func getDefaultTemplateFuncs() libtemplate.FuncMap {
	return libtemplate.FuncMap{
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

func getTemplateFuncs(pluginsDir string) (libtemplate.FuncMap, error) {
	funcs := getDefaultTemplateFuncs()

	var plugins []string
	err := filepath.Walk(
		pluginsDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !strings.HasSuffix(path, ".so") {
				return nil
			}

			plugins = append(plugins, path)

			return nil
		},
	)
	if err != nil {
		if os.IsNotExist(err) {
			return funcs, nil
		}

		return nil, ser.Errorf(
			err, "unable to lookup plugins",
		)
	}

	for _, path := range plugins {
		pluginFuncs, err := loadPlugin(path)
		if err != nil {
			return nil, ser.Errorf(
				err, "can't load plugin: %s", path,
			)
		}

		for name, reference := range *pluginFuncs {
			funcs[name] = reference
		}
	}

	return funcs, nil
}

func loadPlugin(path string) (*libtemplate.FuncMap, error) {
	symbols, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	exports, err := symbols.Lookup("Exports")
	if err != nil {
		return nil, ser.Errorf(
			err, "can't lookup Exports variable",
		)
	}

	funcs, ok := exports.(*libtemplate.FuncMap)
	if !ok {
		return nil, ser.Errorf(
			err, "Exports variable can't be converted to template.FuncMap",
		)
	}

	return funcs, nil
}
