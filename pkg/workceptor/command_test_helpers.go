package workceptor

import (
	"os/exec"
)

// CommandUnit is a test helper type that wraps commandUnit for testing
type CommandUnit struct {
	*commandUnit
}

// TestCommandRunner is a test helper function that exposes the commandRunner function for testing
func (cw *CommandUnit) TestCommandRunner(command string, params string, unitdir string) error {
	return commandRunner(command, params, unitdir)
}

// TestCommandRunner is a test helper function that exposes the commandRunner function for testing
func (cw *commandUnit) TestCommandRunner(command string, params string, unitdir string) error {
	return commandRunner(command, params, unitdir)
}

// TestGetWorkType is a test helper function that exposes the GetWorkType function for testing
func (cfg CommandWorkerCfg) TestGetWorkType() string {
	return cfg.GetWorkType()
}

// TestGetVerifySignature is a test helper function that exposes the GetVerifySignature function for testing
func (cfg CommandWorkerCfg) TestGetVerifySignature() bool {
	return cfg.GetVerifySignature()
}

// TestRun is a test helper function that exposes the Run function for testing
func (cfg CommandWorkerCfg) TestRun() error {
	return cfg.Run()
}

// TestRunCommandRunner is a test helper function that exposes the Run function for testing
func (cfg commandRunnerCfg) TestRun() error {
	return cfg.Run()
}

// TestTermThenKill is a test helper function that exposes the termThenKill function for testing
func TestTermThenKill(cmd *exec.Cmd, doneChan chan bool) {
	termThenKill(cmd, doneChan)
}
