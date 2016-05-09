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
	"github.com/zazab/zhash"
)

const usage = `Gunter 1.3,

Gunter is a configuration system which is created with KISS (Keep It Short and
Simple) principle in mind.

Gunter takes a files and directories from the templates directory, takes a
configuration data from the configuration file written in TOML language, and
then create directories with the same names, renders template files via Go
template engine, and puts result to destination directory.

Of course, gunter will save file permissions including file owner uid/gid of
the copied files and directories.

Usage:
    gunter [-t <tpl>] [-c <config>] [-d <dir>] [-b <dir>] [-l <path>]
    gunter [-t <tpl>] [-c <config>] -r

Options:
    -t <tpl>     Set source templates directory.
                     [default: /var/gunter/templates/]
    -c <config>  Set source file with configuration data.
                     [default: /etc/gunter/config]
    -d <dir>     Set destination directory, where rendered template files
                     and directories will be saved.  [default: /]
    -b <dir>     Set backup directory for storing files, which
                     will be overwriten.
    -l <path>    Set file path, which will be used for logging list of
                     created/overwrited files.
    -r           "Dry Run" mode. Gunter will create the temporary directory,
                     print location and use it as destination directory.
`

func main() {
	args, err := docopt.Parse(usage, nil, true, "1.3", false)
	if err != nil {
		panic(err)
	}

	var (
		configFile   = args["-c"].(string)
		templatesDir = args["-t"].(string)
		destDir      = args["-d"].(string)
		dryRun       = args["-r"].(bool)

		logPath, shouldWriteLogs = args["-l"].(string)

		backupDir, shouldBackup = args["-b"].(string)
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
		templates, config.GetRoot(), tempDir,
	)
	if err != nil {
		log.Fatal(err)
	}

	if dryRun {
		fmt.Printf(
			"configuration files are saved into temporary directory %s\n",
			tempDir,
		)

		os.Exit(0)
	}

	err = moveFiles(
		tempDir, destDir,
		logPath, shouldWriteLogs,
		backupDir, shouldBackup,
	)
	if err != nil {
		log.Fatal(err)
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
) error {
	walker := PlaceWalker{
		sourceDir:    sourceDir,
		destDir:      destDir,
		shouldBackup: shouldBackup,
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
			return fmt.Errorf("can't write log file %s: %s", logPath, err)
		}
	}

	err = os.RemoveAll(sourceDir)
	if err != nil {
		return fmt.Errorf("can't remove %s: %s", sourceDir, err)
	}

	return nil
}
