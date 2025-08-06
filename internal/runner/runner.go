package runner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type ExecError struct {
	Err      error
	ExitCode int
	Output   string
}

func (r *ExecError) Error() string {
	return fmt.Sprintf("status %d: err %v, output: %s", r.ExitCode, r.Err, r.Output)
}

func (r *ExecError) Unwrap() error {
	return r.Err
}

type runner struct{}

func NewWithDefaultGitlabBackupCLIPath() (runner, error) {
	return runner{}, nil
}

func (r runner) Run(ctx context.Context, command string, args ...string) error {
	fullCommand := fmt.Sprintf("%s %s", command, strings.Join(args, " "))

	log.Printf("Running shell command: bash -c '%s'", fullCommand)

	cmd := exec.CommandContext(ctx, "bash", "-c", fullCommand)

	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitError *exec.ExitError

		switch {
		case errors.As(err, &exitError):
			return fmt.Errorf("exec command: %w",
				&ExecError{
					Err:      err,
					Output:   string(out),
					ExitCode: exitError.ExitCode(),
				},
			)
		default:
			return fmt.Errorf("exec command: %w", err)
		}
	}

	return nil
}
