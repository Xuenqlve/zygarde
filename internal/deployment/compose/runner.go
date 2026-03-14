package compose

import (
	"context"
	"os/exec"
)

// CommandRunner runs external commands for compose lifecycle operations.
type CommandRunner interface {
	Run(ctx context.Context, workdir string, name string, args ...string) (string, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, workdir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	return string(output), err
}
