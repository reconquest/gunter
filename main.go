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

func getTemplates(templatesDir string) ([]templateItem, error) {
	storage, err := NewTemplateStorage(templatesDir)
	if err != nil {
		return nil, err
	}

	return storage.GetItems()
}

func compileTemplates(
	templates []templateItem,
	config map[string]interface{},
	destDir string,
) (err error) {
	for _, template := range templates {
		switch {
		case template.Mode().IsRegular():
			if strings.HasSuffix(template.RelativePath(), ".template") {
				err = compileTemplateFile(template, destDir, config)
			} else {
				err = copyTemplateFile(template, destDir)
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

	return applyPermissions(dirPath, template)
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

	// strict mode
	tpl.Option("missingkey=error")

	err = tpl.Execute(compiledFile, config)
	if err != nil {
		return err
	}

	return applyPermissions(filePath, template)
}

func copyTemplateFile(
	template templateItem, destDir string,
) error {
	dest := filepath.Join(destDir, template.RelativePath())
	err := copyFile(template.FullPath(), dest, template.Mode())

	if err != nil {
		return err
	}

	return applyPermissions(dest, template)
}

func copyFile(sourcePath, destPath string, mode os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	defer sourceFile.Close()

	destFile, err := os.OpenFile(
		destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode,
	)
	if err != nil {
		return err
	}

	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Close()
}

func getTempDir() (string, error) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "gunter")
	if err != nil {
		return "", err
	}

	tempDir = tempDir + "/"

	return tempDir, nil
}

func applyPermissions(path string, fileinfo os.FileInfo) error {
	err := os.Chown(
		path,
		int(fileinfo.Sys().(*syscall.Stat_t).Uid),
		int(fileinfo.Sys().(*syscall.Stat_t).Gid),
	)
	if err != nil {
		return err
	}

	err = os.Chmod(path, fileinfo.Mode())

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

func moveFiles(
	sourceDir, destDir,
	logPath string, shouldWriteLogs bool,
	backupDir string, shouldBackup bool,
) error {
	walker := CopyWalker{
		sourceDir: sourceDir,
		destDir:   destDir,
		backup:    shouldBackup,
		backupDir: backupDir,
	}

	err := filepath.Walk(sourceDir, walker.Walk)
	if err != nil {
		return err
	}

	if shouldWriteLogs {
		err = ioutil.WriteFile(
			logPath, []byte(strings.Join(walker.modified, "\n")), 0644,
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
