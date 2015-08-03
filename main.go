package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	libtemplate "text/template"

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
		configFile   = args["-c"].(string)
		templatesDir = args["-t"].(string)
		destDir      = args["-d"].(string)
		dryRun       = args["-r"].(bool)
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
		destDir, err = getTempDir()
		if err != nil {
			log.Fatal(err)
		}
	}

	err = compileTemplates(templates, destDir, config.GetRoot())
	if err != nil {
		log.Fatal(err)
	}

	if dryRun {
		log.Printf(
			"configuration files are saved into temporary directory %s\n",
			destDir,
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
	templates []templateItem, destDir string, config map[string]interface{},
) (err error) {
	destDir = strings.TrimRight(destDir, "/") + "/"

	for _, template := range templates {
		switch {
		case template.Mode().IsRegular():
			if strings.HasSuffix(template.RelativePath(), ".template") {
				err = compileTemplateFile(template, destDir, config)
			} else {
				err = copyFile(template, destDir)
			}

		case template.Mode().IsDir():
			if strings.HasSuffix(template.RelativePath(), ".template") {
				template, templates = removeDotTemplateDirectorySuffix(
					template, templates,
				)
			}

			err = compileTemplateDir(template, destDir)

		default:
			err = fmt.Errorf(
				"file '%s' has unsupported file type",
				template.RelativePath(),
			)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func removeDotTemplateDirectorySuffix(
	target templateItem, templates []templateItem,
) (templateItem, []templateItem) {
	newRelativePath := strings.TrimSuffix(target.RelativePath(), ".template")

	for index, template := range templates {
		if strings.HasPrefix(template.RelativePath(), target.RelativePath()) {
			template.SetRelativePath(
				newRelativePath + strings.TrimPrefix(
					template.RelativePath(),
					target.RelativePath(),
				),
			)

			templates[index] = template
		}
	}

	target.SetRelativePath(newRelativePath)

	return target, templates
}

func compileTemplateDir(template templateItem, destDir string) error {
	dirPath := filepath.Join(destDir, template.RelativePath())
	err := os.Mkdir(dirPath, template.Mode())
	if err != nil && !os.IsExist(err) {
		return err
	}

	return applyTemplatePermissions(dirPath, template)
}

func compileTemplateFile(
	template templateItem, destDir string, config map[string]interface{},
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

	filePath := filepath.Join(
		destDir, strings.TrimSuffix(template.RelativePath(), ".template"),
	)

	compiledFile, err := os.OpenFile(
		filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, template.Mode(),
	)
	if err != nil {
		return err
	}

	defer compiledFile.Close()

	err = tpl.Execute(compiledFile, config)
	if err != nil {
		return err
	}

	return applyTemplatePermissions(filePath, template)
}

func copyFile(
	template templateItem, destDir string,
) error {
	sourceFile, err := os.Open(template.FullPath())
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	filePath := filepath.Join(destDir, template.RelativePath())

	destFile, err := os.OpenFile(
		filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, template.Mode(),
	)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return applyTemplatePermissions(filePath, template)
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
