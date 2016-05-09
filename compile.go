package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	libtemplate "text/template"
)

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

	tpl := libtemplate.New(template.RelativePath())

	// strict mode
	tpl.Option("missingkey=error")

	tpl.Funcs(getTemplateFuncs())

	tpl, err = tpl.Parse(
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
