package utils

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.Chmod(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	return nil
}

func CopyDir(src, dst string) error {
	if strings.HasPrefix(dst, src) {
		return fmt.Errorf("%s is inside %s, causing infinite recursion", dst, src)
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func FormatPermissions(mode os.FileMode) string {
	permissions := []byte{'-', '-', '-', '-', '-', '-', '-', '-', '-', '-'}

	if mode.IsDir() {
		permissions[0] = 'd'
	}

	rwxBits := []os.FileMode{0400, 0200, 0100, 0040, 0020, 0010, 0004, 0002, 0001}
	rwxChars := []byte{'r', 'w', 'x'}

	for i, bit := range rwxBits {
		if mode&bit != 0 {
			permissions[i+1] = rwxChars[i%3]
		}
	}

	specialBits := []struct {
		mask     os.FileMode
		position int
		execPos  int
		char     byte
	}{
		{os.ModeSetuid, 3, 3, 's'},
		{os.ModeSetgid, 6, 6, 's'},
		{os.ModeSticky, 9, 9, 't'},
	}

	for _, sp := range specialBits {
		if mode&sp.mask != 0 {
			if permissions[sp.execPos] == 'x' {
				permissions[sp.position] = sp.char
			} else {
				permissions[sp.position] = sp.char - ('x' - 'X')
			}
		}
	}

	return string(permissions)
}

func GetFileInfo(info os.FileInfo, path string) (permString, permOctal, owner, group string, err error) {
	permString = FormatPermissions(info.Mode())
	permOctal = fmt.Sprintf("%o", info.Mode().Perm())

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", "", "", "", fmt.Errorf("failed to get system stat information")
	}

	uidStr := strconv.Itoa(int(stat.Uid))
	gidStr := strconv.Itoa(int(stat.Gid))

	ownerInfo, err := user.LookupId(uidStr)
	if err != nil {
		return "", "", "", "", err
	}
	groupInfo, err := user.LookupGroupId(gidStr)
	if err != nil {
		return "", "", "", "", err
	}

	return permString, permOctal, ownerInfo.Username, groupInfo.Name, nil
}
