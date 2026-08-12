package lifecycle

import (
	"bytes"
	"errors"
	"os/exec"

	"taskai/internal/backgroundprocess"
)

type GitInvocation struct {
	Directory string
	Arguments []string
}

type GitExecutor interface {
	Run(GitInvocation) (CommandResult, error)
}

type GitExecutorFunc func(GitInvocation) (CommandResult, error)

func (function GitExecutorFunc) Run(invocation GitInvocation) (CommandResult, error) {
	return function(invocation)
}

type GitCommandExitError struct {
	Code  int
	Cause error
}

func (err *GitCommandExitError) Error() string {
	if err.Cause != nil {
		return err.Cause.Error()
	}
	return "Git 命令退出失败"
}

func (err *GitCommandExitError) Unwrap() error {
	return err.Cause
}

type DirectGitExecutor struct{}

func NewDirectGitExecutor() *DirectGitExecutor {
	return &DirectGitExecutor{}
}

func (executor *DirectGitExecutor) Run(invocation GitInvocation) (CommandResult, error) {
	process := exec.Command("git", invocation.Arguments...)
	backgroundprocess.Configure(process)
	process.Dir = invocation.Directory
	var output bytes.Buffer
	var standardError bytes.Buffer
	process.Stdout = &output
	process.Stderr = &standardError
	err := process.Run()
	if err == nil {
		return CommandResult{Output: output.Bytes(), StandardError: standardError.Bytes()}, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return CommandResult{Output: output.Bytes(), StandardError: standardError.Bytes()}, &GitCommandExitError{Code: exitError.ExitCode(), Cause: err}
	}
	return CommandResult{Output: output.Bytes(), StandardError: standardError.Bytes()}, err
}

func isGitRemoteBranchMissing(err error) bool {
	var exitError *GitCommandExitError
	return errors.As(err, &exitError) && exitError.Code == 2
}
