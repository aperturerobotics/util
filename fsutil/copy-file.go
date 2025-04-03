package fsutil

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// CopyFile copies the contents from src to dst.
func CopyFile(dst, src string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
	}
	return err
}

// MoveFile moves the contents from src to dst.
func MoveFile(dst, src string, perm os.FileMode) error {
	if err := CopyFile(dst, src, perm); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return err
	}
	return nil
}

// CopyFileToDir copies the file to the dir maintaining the filename.
func CopyFileToDir(dstDir, src string, perm os.FileMode) error {
	_, srcFilename := filepath.Split(src)
	return CopyFile(filepath.Join(dstDir, srcFilename), src, perm)
}

// MoveFileToDir moves the contents from src to dstDir maintaining the filename.
func MoveFileToDir(dstDir, src string, perm os.FileMode) error {
	_, srcFilename := filepath.Split(src)
	return MoveFile(filepath.Join(dstDir, srcFilename), src, perm)
}

// CopyRecursive copies regular files & directories from src to dest.
//
// Calls the callback with the absolute path to the source file.
// Ignore not-exist src dir by doing nothing.
func CopyRecursive(dstDir, src string, cb fs.WalkDirFunc) error {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(src, func(srcPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fi, err := info.Info()
		if err != nil {
			return err
		}

		srcRel, err := filepath.Rel(src, srcPath)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, srcRel)
		dstParent := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstParent, 0o755); err != nil {
			return err
		}
		if info.Type().IsRegular() {
			if err := CopyFile(dstPath, srcPath, fi.Mode().Perm()); err != nil {
				return &fs.PathError{
					Op:   "copy",
					Path: srcRel,
					Err:  err,
				}
			}
		} else if info.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
		} else if info.Type()&fs.ModeSymlink != 0 {
			dstLink, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}

			if err := os.Symlink(dstLink, dstPath); err != nil {
				return err
			}
		}

		if cb != nil {
			if err := cb(srcPath, info, err); err != nil {
				return err
			}
		}

		return nil
	})
}
