package main

import (
	"fmt"
	libtemplate "html/template"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/docopt/docopt-go"
	"github.com/zazab/zhash"
)

const usage = `Gunter 1.0,

Gunter is a configuration system which is created with KISS (Keep It Short and
Simple) principle in mind.

Gunter takes a files and directories from the templates directory, takes a
configuration data from the configuration file written in TOML language, and
then create directories with the same names, renders template files via Go
template engine, and puts result to destination directory.

Of course, gunter will save file permissions including file owner uid/gid of
the copied files and directories.

Usage:
    gunter [-t <tpl>] [-c <config>] [-d <dir>]
    gunter [-t <tpl>] [-c <config>] -r

Options:
    -t <tpl>     Set source templates directory.
                 [default: /var/gunter/templates/]
    -c <config>  Set source file with configuration data.
                 [default: /etc/gunter/config]
    -d <dir>     Set destination directory, where rendered template files and
                 directories will be saved.  [default: /]
    -r           "Dry Run" mode. Gunter will create the temporary directory,
                 print location and use it as destination directory.
`

func main() {
	args, _ := docopt.Parse(usage, nil, true, "1.0", false)

	var (
		configFile     = args["-c"].(string)
		templatesDir   = args["-t"].(string)
		destinationDir = args["-d"].(string)
		dryRun         = args["-r"].(bool)
	)

	config, err := getConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	templates, err := getTemplates(templatesDir)
	if err != nil {
		log.Fatal(err)
	}

	if dryRun {
		destinationDir, err = getTempDir()
		if err != nil {
			log.Fatal(err)
		}
	}

	err = compileTemplates(templates, destinationDir, config.GetRoot())
	if err != nil {
		log.Fatal(err)
	}

	if dryRun {
		log.Printf(
			"configuration files are saved into temporary directory %s\n",
			destinationDir,
		)
	}
}

func getTemplates(templatesDir string) ([]templateItem, error) {
	storage, err := NewTemplateStorage(templatesDir)
	if err != nil {
		return nil, err
	}

	return storage.GetItems()
}

func compileTemplates(
	templates []templateItem,
	destinationDir string,
	config map[string]interface{},
) (err error) {
	destinationDir = strings.TrimRight(destinationDir, "/") + "/"

	for _, template := range templates {
		switch {
		case template.Mode().IsRegular():
			err = compileTemplateFile(template, destinationDir, config)

		case template.Mode()&os.ModeDir == os.ModeDir:
			err = compileTemplateDir(template, destinationDir)

		default:
			err = fmt.Errorf(
				"file '%s' has unsupported file type",
				template.RelativePath(),
			)
		}

		if err != nil {
			return err
		}

		err = applyTemplatePermissions(
			destinationDir+template.RelativePath(),
			template,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func compileTemplateDir(
	template templateItem, destinationDir string,
) error {
	err := os.Mkdir(
		destinationDir+template.RelativePath(), template.Mode(),
	)

	if err != nil && !os.IsExist(err) {
		return err
	}

	err = applyTemplatePermissions(
		destinationDir+template.RelativePath(),
		template,
	)
	if err != nil {
		return err
	}

	return nil
}

func compileTemplateFile(
	template templateItem,
	destinationDir string,
	config map[string]interface{},
) error {
	templateContents, err := ioutil.ReadFile(template.FullPath())
	if err != nil {
		return err
	}

	tpl, err := libtemplate.New(template.RelativePath()).Parse(
		tplStripWhitespaces(string(templateContents)),
	)
	if err != nil {
		return err
	}

	compiledFile, err := os.OpenFile(
		destinationDir+template.RelativePath(),
		os.O_CREATE|os.O_WRONLY,
		template.Mode(),
	)
	if err != nil {
		return err
	}

	defer compiledFile.Close()

	err = tpl.Execute(compiledFile, config)
	if err != nil {
		return err
	}

	return nil
}

func getTempDir() (string, error) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "gunter")
	if err != nil {
		return "", err
	}

	tempDir = tempDir + "/"

	return tempDir, nil
}

func applyTemplatePermissions(path string, template templateItem) error {
	err := os.Chown(
		path,
		int(template.Sys().(*syscall.Stat_t).Uid),
		int(template.Sys().(*syscall.Stat_t).Gid),
	)
	if err != nil {
		return err
	}

	err = os.Chmod(path, template.Mode())

	return err
}

func getConfig(path string) (zhash.Hash, error) {
	configData := map[string]interface{}{}
	_, err := toml.DecodeFile(path, &configData)
	if err != nil {
		return zhash.Hash{}, err
	}

	return zhash.HashFromMap(configData), nil
}
