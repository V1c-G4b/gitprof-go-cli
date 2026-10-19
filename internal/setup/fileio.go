package setup

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func readOptional(path string) (string, bool, os.FileMode, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, 0, nil
	}
	if err != nil {
		return "", false, 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, 0, err
	}
	return string(data), true, info.Mode().Perm(), nil
}

func linkTarget(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return path, nil
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real, nil
	}
	cur := path
	for range 40 {
		dest, err := os.Readlink(cur)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(filepath.Dir(cur), dest)
		}
		info, err := os.Lstat(dest)
		if errors.Is(err, fs.ErrNotExist) {
			return dest, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			return dest, nil
		}
		cur = dest
	}
	return "", fmt.Errorf("links demais em %s", path)
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTSUP) && !errors.Is(err, errors.ErrUnsupported) {
		return fmt.Errorf("fsync de %s: %w", dir, err)
	}
	return nil
}

func writeAtomic(path, content string, mode os.FileMode) error {
	path, err := linkTarget(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("criar temporário para %s: %w", path, err)
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err := io.WriteString(tmp, content); err != nil {
		return fmt.Errorf("escrever %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renomear para %s: %w", path, err)
	}
	ok = true
	return syncDir(dir)
}

func backup(path string) error {
	path, err := linkTarget(path)
	if err != nil {
		return err
	}
	data, exists, mode, err := readOptional(path)
	if err != nil || !exists {
		return err
	}
	dst := path + BackupSuffix
	if fileExists(dst) {
		return nil
	}
	return writeAtomic(dst, data, mode)
}

func resolvePath(p string) string {
	p = filepath.Clean(p)
	cur, rest := p, ""
	for {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(r, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}
