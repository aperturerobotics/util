package gitcmd

import (
	"bufio"
	"os/exec"

	util_bufio "github.com/aperturerobotics/util/bufio"
)

// ListGitFiles runs "git ls-files" to list all files in a Git workdir including
// modified and untracked files, but not including deleted files.
//
// Returns the paths in ascending sorted order, with format "dir/file.txt".
func ListGitFiles(workDir string) ([]string, error) {
	var files []string

	// Exec git ls-files with null terminated entries.
	cmd := exec.Command(
		"git",
		"ls-files",
		"-z",
		"--exclude-standard",
		"--others",
		"--cached",
	)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Split(util_bufio.SplitOnNul)

	for scanner.Scan() {
		file := scanner.Text()
		files = append(files, file)
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return files, nil
}
