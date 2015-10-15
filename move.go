package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type CopyWalker struct {
	modified  []string
	sourceDir string
	destDir   string
	backup    bool
	backupDir string
}

func (walker CopyWalker) Walk(
	sourcePath string, sourceInfo os.FileInfo, err error,
) error {
	relativePath := strings.TrimPrefix(sourcePath, walker.sourceDir)
	if relativePath == "/" {
		return nil
	}

	var (
		destPath   = filepath.Join(walker.destDir, relativePath)
		backupPath = filepath.Join(walker.backupDir, relativePath)
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
		if !sourceInfo.IsDir() && !destInfo.IsDir() {
			sourceHash, err := getHash(sourcePath)
			if err != nil {
				return err
			}

			destHash, err := getHash(destPath)
			if err != nil {
				return err
			}

			if sourceHash == destHash {
				// should not copy files, if they has same content (hash sum)
				return nil
			}
		}

		if walker.backup {
			// copying file/directory from destination to backup
			err = walker.copy(destPath, backupPath, destInfo)
			if err != nil {
				return err
			}
		}

		if sourceInfo.IsDir() != destInfo.IsDir() {
			err = os.RemoveAll(destPath)
			if err != nil {
				return fmt.Errorf(
					"can't delete %s: %s", destPath, err,
				)
			}
		}
	}

	err = walker.copy(sourcePath, destPath, sourceInfo)
	if err != nil {
		return err
	}

	return nil

}

func (walker CopyWalker) copy(
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
