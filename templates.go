package main

import (
	"os"
	"path/filepath"
	"strings"
)

type templateStorage struct {
	dir         string
	files       []templateItem
	initialized bool
}

type templateItem struct {
	os.FileInfo
	fullPath     string
	relativePath string
}

func NewTemplateStorage(templatesDir string) (*templateStorage, error) {
	templatesDir = strings.TrimRight(templatesDir, "/") + "/"

	return &templateStorage{
		dir:   templatesDir,
		files: []templateItem{},
	}, nil
}

func (storage *templateStorage) GetItems() ([]templateItem, error) {
	if !storage.initialized {
		err := filepath.Walk(storage.dir, storage.walkDir)
		if err != nil {
			return []templateItem{}, err
		}

		storage.initialized = true
	}

	return storage.files, nil
}

func (storage *templateStorage) walkDir(
	fullPath string, info os.FileInfo, err error,
) error {
	if err != nil {
		return err
	}

	relativePath := strings.TrimPrefix(fullPath, storage.dir)

	// skip root dir
	if relativePath == "" {
		return nil
	}

	storage.add(info, fullPath, relativePath)

	return nil
}

func (storage *templateStorage) add(
	info os.FileInfo,
	fullPath, relativePath string,
) {
	storage.files = append(
		storage.files,
		templateItem{
			info,
			fullPath,
			relativePath,
		},
	)
}

// methods named by analogue with os.FileInfo Name()/Size()/Mode()
func (item *templateItem) RelativePath() string {
	return item.relativePath
}

func (item *templateItem) FullPath() string {
	return item.fullPath
}

func (item *templateItem) SetRelativePath(path string) {
	item.relativePath = path
}

func (item *templateItem) SetFullPath(path string) {
	item.fullPath = path
}
