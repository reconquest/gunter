package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/seletskiy/hierr"

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
				if err != nil {
					return hierr.Errorf(
						err,
						"can't compile template file %s (%s)",
						template.RelativePath(), template.FullPath(),
					)
				}
			} else {
				err = copyTemplateFile(template, destDir)
				if err != nil {
					return hierr.Errorf(
						err,
						"can't copy file %s (%s)",
						template.RelativePath(), template.FullPath(),
					)
				}
			}

		case template.Mode().IsDir():
			if strings.HasSuffix(template.RelativePath(), ".template") {
				template, templates = removeDotTemplateDirectorySuffix(
					template, templates,
				)
			}

			err = compileTemplateDir(template, destDir)
			if err != nil {
				return hierr.Errorf(
					err,
					"can't compile template directory %s (%s)",
					template.RelativePath(), template.FullPath(),
				)
			}

		default:
			return fmt.Errorf(
				"file '%s' has unsupported file type",
				template.RelativePath(),
			)
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
		return hierr.Errorf(
			err, "can't create directory %s", dirPath,
		)
	}

	err = applyPermissions(dirPath, template)
	if err != nil {
		return hierr.Errorf(
			err, "can't apply file permissions for directory %s", dirPath,
		)
	}

	return nil
}

func compileTemplateFile(
	template templateItem, destDir string, config map[string]interface{},
) error {
	templateContents, err := ioutil.ReadFile(template.FullPath())
	if err != nil {
		return hierr.Errorf(
			err, "can't read file %s", template.FullPath(),
		)
	}

	tpl := libtemplate.New(template.RelativePath())

	// strict mode
	tpl.Option("missingkey=error")

	tpl.Funcs(getTemplateFuncs())

	tpl, err = tpl.Parse(
		tplStripWhitespaces(string(templateContents)),
	)
	if err != nil {
		return hierr.Errorf(
			err, "can't parse template",
		)
	}

	filePath := filepath.Join(
		destDir, strings.TrimSuffix(template.RelativePath(), ".template"),
	)

	compiledFile, err := os.OpenFile(
		filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, template.Mode(),
	)
	if err != nil {
		return hierr.Errorf(
			err, "can't open file %s", filePath,
		)
	}

	defer compiledFile.Close()

	err = tpl.Execute(compiledFile, config)
	if err != nil {
		return hierr.Errorf(
			err, "can't execute template",
		)
	}

	err = applyPermissions(filePath, template)
	if err != nil {
		return hierr.Errorf(
			err, "can't apply file permissions for file %s", filePath,
		)
	}

	return nil
}

func copyTemplateFile(
	template templateItem, destDir string,
) error {
	dest := filepath.Join(destDir, template.RelativePath())
	err := copyFile(template.FullPath(), dest, template.Mode())
	if err != nil {
		return hierr.Errorf(
			err, "can't copy file %s to %s", template.FullPath(), dest,
		)
	}

	err = applyPermissions(dest, template)
	if err != nil {
		return hierr.Errorf(
			err, "can't apply file permissions for file %s", dest,
		)
	}

	return nil
}

func copyFile(sourcePath, destPath string, mode os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return hierr.Errorf(
			err, "can't open %s", sourcePath,
		)
	}

	defer sourceFile.Close()

	destFile, err := os.OpenFile(
		destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode,
	)
	if err != nil {
		return hierr.Errorf(
			err, "can't open %s", destPath,
		)
	}

	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return hierr.Errorf(
			err, "can't copy data",
		)
	}

	return nil
}

func getTempDir() (string, error) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "gunter")
	if err != nil {
		return "", hierr.Errorf(
			err, "can't get temporary directory",
		)
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
		return hierr.Errorf(
			err, "can't make chown",
		)
	}

	err = os.Chmod(path, fileinfo.Mode())
	if err != nil {
		return hierr.Errorf(
			err, "can't make chmod",
		)
	}

	return nil
}
