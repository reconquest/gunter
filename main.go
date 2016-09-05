package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/docopt/docopt-go"
	"github.com/seletskiy/hierr"
	"github.com/zazab/zhash"
)

var (
	version = `1.4`
	usage   = `gunter ` + version + `

gunter is a configuration system which is created with KISS (Keep It Short and
Simple) principle in mind.

gunter takes a files and directories from the templates directory, takes a
configuration data from the configuration file written in TOML language, and
then create directories with the same names, renders template files via Go
template engine, and puts result to destination directory.

Of course, gunter will save file permissions including file owner uid/gid of
the copied files and directories.

Usage:
    gunter [-t <dir>] [-c <config>] [-d <dir>] [-l <path>] [-n] [-b <dir>]
    gunter [-t <dir>] [-c <config>] [-d <dir>] [-l <path>] [-n] -r

Options:
  -t --templates <path>  Set source templates directory.
                          [default: /var/gunter/templates/]
  -c --config <config>   Set source file with configuration data.
                          [default: /etc/gunter/config]
  -n --non-strict        Do not abort on missing values in the templates.
                         In-place value replacements will be rendered as
                          '<no value>', range values will be rendered empty.
  -d --target <dir>      Set destination directory, where rendered template
                          files and directories will be saved.
                          [default: /]
  -b --backup <dir>      Set backup directory for storing files, which
                          will be overwriten.
  -l --log <path>        Set file path, which will be used for logging list
                          of created/overwrited files.
  -r --dry-run           "Dry Run" mode. gunter will create the temporary
                          directory, print location and use it as destination
                          directory.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	var (
		configFile               = args["--config"].(string)
		templatesDir             = args["--templates"].(string)
		destDir                  = args["--target"].(string)
		dryRun                   = args["--dry-run"].(bool)
		logPath, shouldWriteLogs = args["--log"].(string)
		backupDir, shouldBackup  = args["--backup"].(string)
		nonStrict                = CompileMode(args["--non-strict"].(bool))
	)

	config, err := getConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	templates, err := getTemplates(templatesDir)
	if err != nil {
		log.Fatal(err)
	}

	tempDir, err := getTempDir()
	if err != nil {
		log.Fatal(err)
	}

	err = compileTemplates(
		templates, config.GetRoot(), tempDir, nonStrict,
	)
	if err != nil {
		log.Fatal(err)
	}

	if dryRun {
		fmt.Fprintf(
			os.Stderr,
			"configuration files are saved into temporary directory %s\n",
			tempDir,
		)
	}

	err = moveFiles(
		tempDir, destDir,
		logPath, shouldWriteLogs,
		backupDir, shouldBackup,
		dryRun,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConfig(path string) (zhash.Hash, error) {
	configData := map[string]interface{}{}
	_, err := toml.DecodeFile(path, &configData)
	if err != nil {
		return zhash.Hash{}, err
	}

	return zhash.HashFromMap(configData), nil
}

func getTemplates(templatesDir string) ([]templateItem, error) {
	storage, err := NewTemplateStorage(templatesDir)
	if err != nil {
		return nil, err
	}

	return storage.GetItems()
}

func moveFiles(
	sourceDir, destDir,
	logPath string, shouldWriteLogs bool,
	backupDir string, shouldBackup bool,
	dryRun bool,
) error {
	walker := PlaceWalker{
		sourceDir:    sourceDir,
		destDir:      destDir,
		shouldBackup: shouldBackup,
		dryRun:       dryRun,
		backupDir:    backupDir,
	}

	err := filepath.Walk(sourceDir, walker.Place)
	if err != nil {
		return err
	}

	if shouldWriteLogs {
		err = ioutil.WriteFile(
			logPath, []byte(strings.Join(walker.placed, "\n")+"\n"), 0644,
		)
		if err != nil {
			return hierr.Errorf(
				err, "can't write log",
			)
		}
	}

	if !dryRun {
		err = os.RemoveAll(sourceDir)
		if err != nil {
			return hierr.Errorf(
				err, "can't remove %s", sourceDir,
			)
		}
	}

	return nil
}
