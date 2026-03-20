package services

import "syscall"

// StatFS holds filesystem statistics.
type StatFS struct {
	Blocks uint64
	Bfree  uint64
	Bsize  int64
}

// statfs returns filesystem statistics for the given path.
func statfs(path string, stat *StatFS) error {
	var s syscall.Statfs_t
	if err := syscall.Statfs(path, &s); err != nil {
		return err
	}
	stat.Blocks = s.Blocks
	stat.Bfree = s.Bfree
	stat.Bsize = s.Bsize
	return nil
}
