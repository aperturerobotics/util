package exec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

// NewCmd builds a new exec cmd with defaults.
func NewCmd(proc string, args ...string) *exec.Cmd {
	ecmd := exec.Command(proc, args...)
	ecmd.Env = make([]string, len(os.Environ()))
	copy(ecmd.Env, os.Environ())
	ecmd.Stderr = os.Stderr
	ecmd.Stdout = os.Stdout
	return ecmd
}

// StartAndWait runs the given process and waits for ctx or process to complete.
func StartAndWait(ctx context.Context, le *logrus.Entry, ecmd *exec.Cmd) error {
	if ecmd.Process == nil {
		le.Debugf("exec: %s", ecmd.String())
		if err := ecmd.Start(); err != nil {
			return err
		}
	}

	outErr := make(chan error, 1)
	go func() {
		outErr <- ecmd.Wait()
	}()
	select {
	case <-ctx.Done():
		_ = ecmd.Process.Kill()
		<-outErr
		return ctx.Err()
	case err := <-outErr:
		le := le.WithField("exit-code", ecmd.ProcessState.ExitCode())
		if err != nil {
			le.WithError(err).Debug("process exited with error")
		} else {
			le.Debug("process exited")
		}
		return err
	}
}

// ExecCmd runs the command and collects the log output.
func ExecCmd(le *logrus.Entry, cmd *exec.Cmd) error {
	var stderrBuf bytes.Buffer

	goLogger := le.WriterLevel(logrus.DebugLevel)
	cmd.Stderr = io.MultiWriter(&stderrBuf, goLogger)
	le.
		WithField("work-dir", cmd.Dir).
		Debugf("running command: %s", cmd.String())

	err := cmd.Run()
	if err != nil && (strings.HasPrefix(err.Error(), "exit status") || strings.HasPrefix(err.Error(), "err: exit status")) {
		stderrLines := strings.Split(stderrBuf.String(), "\n")
		errMsg := stderrLines[len(stderrLines)-1]
		if len(errMsg) == 0 && len(stderrLines) > 1 {
			errMsg = stderrLines[len(stderrLines)-2]
		}
		err = errors.New(errMsg)
	}
	return err
}
