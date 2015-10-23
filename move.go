package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type PlaceWalker struct {
	placed       []string
	sourceDir    string
	destDir      string
	shouldBackup bool
	backupDir    string
}

func (walker *PlaceWalker) Place(
	sourcePath string, sourceInfo os.FileInfo, err error,
) error {
	relativePath := strings.TrimPrefix(sourcePath, walker.sourceDir)
	if relativePath == "" {
		return nil
	}

	var (
		destPath   = filepath.Join(walker.destDir, relativePath)
		destExists = true
	)

	destInfo, err := os.Stat(destPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		destExists = false
	}

	if destExists {
		same, err := compare(sourcePath, destPath, sourceInfo, destInfo)
		if err != nil {
			return err
		}

		if same {
			return nil
		}

		if walker.shouldBackup {
			err = walker.backup(destPath, destInfo, relativePath)
			if err != nil {
				return err
			}
		}

		if sourceInfo.IsDir() != destInfo.IsDir() {
			if destInfo.IsDir() {
				empty, err := isEmpty(destPath)
				if err != nil {
					return err
				}

				if !empty {
					return fmt.Errorf(
						"destination path %s is a directory, "+
							"but source path %s (%s) is a file, destination "+
							"directory can't be overwrited, because "+
							"is not empty",
						destPath, sourcePath, relativePath,
					)
				}
			}

			err = os.RemoveAll(destPath)
			if err != nil {
				return fmt.Errorf(
					"can't delete %s: %s", destPath, err,
				)
			}
		}
	}

	err = walker.place(sourcePath, destPath, sourceInfo)
	if err != nil {
		return err
	}

	walker.placed = append(walker.placed, "/"+relativePath)

	return nil
}

func (walker *PlaceWalker) place(
	sourcePath, destPath string, sourceInfo os.FileInfo,
) error {
	if sourceInfo.IsDir() {
		err := os.MkdirAll(destPath, sourceInfo.Mode())
		if err != nil {
			return err
		}
	} else {
		err := copyFile(sourcePath, destPath, sourceInfo.Mode())
		if err != nil {
			return fmt.Errorf(
				"can't copy file %s to %s: %s",
				sourcePath, destPath, err,
			)
		}
	}

	err := applyPermissions(destPath, sourceInfo)
	if err != nil {
		return err
	}

	return nil
}

func (walker *PlaceWalker) backup(
	destPath string, destInfo os.FileInfo, relativePath string,
) error {
	dirs := strings.Split(relativePath, "/")

	if len(dirs) > 1 {
		if destInfo.IsDir() {
			dirs = append([]string{}, dirs[:len(dirs)-1]...)
		}

		for index, _ := range dirs {
			subdirs := strings.Join(dirs[:index+1], "/")

			subDestPath := filepath.Join(walker.destDir, subdirs)
			subDestInfo, err := os.Stat(subDestPath)
			if err != nil {
				return err
			}

			subBackupPath := filepath.Join(walker.backupDir, subdirs)

			err = walker.place(subDestPath, subBackupPath, subDestInfo)
			if err != nil {
				return err
			}
		}
	}

	backupPath := filepath.Join(walker.backupDir, relativePath)

	return walker.place(destPath, backupPath, destInfo)
}

func getHash(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	hasher := md5.New()
	hasher.Write(data)

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func compareFileModes(src, dst os.FileInfo) bool {
	srcStat := src.Sys().(*syscall.Stat_t)
	dstStat := src.Sys().(*syscall.Stat_t)

	return src.Mode() == dst.Mode() &&
		srcStat.Uid == dstStat.Uid &&
		srcStat.Gid == dstStat.Gid
}

func compare(
	sourcePath, destPath string, sourceInfo, destInfo os.FileInfo,
) (bool, error) {
	if sourceInfo.IsDir() != destInfo.IsDir() {
		return false, nil
	}

	sameModes := compareFileModes(sourceInfo, destInfo)
	if !sameModes {
		return false, nil
	}

	if sameModes && sourceInfo.IsDir() && destInfo.IsDir() {
		return true, nil
	}

	sourceHash, err := getHash(sourcePath)
	if err != nil {
		return false, err
	}

	destHash, err := getHash(destPath)
	if err != nil {
		return false, err
	}

	if sourceHash == destHash {
		return true, nil
	}

	return false, nil
}

func isEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}

	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}
