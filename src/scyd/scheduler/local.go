package scheduler

import (
	"context"
	"io"
	"os/exec"
	"time"
)

type LocalRunner struct {
}

func NewLocalRunner() (l *LocalRunner) {
	return &LocalRunner{}
}

func (l *LocalRunner) Close() {
	// Does nothing, but required by scheduler
}

func (l *LocalRunner) RunWithWriters(commandLine string, timeout int, sudo bool, stdout_f io.Writer, stderr_f io.Writer) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	shell_command := []string{"/bin/bash", "-c", commandLine}
	command := exec.CommandContext(ctx, shell_command[0], shell_command[1:]...)
	command.Stdout = stdout_f
	command.Stderr = stderr_f
	err = command.Run()
	return err
}
