package util

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestRunCmd(t *testing.T) {
	testCases := []struct {
		name        string
		msg         string
		cmd         string
		args        []string
		expectedErr error
	}{
		{
			name:        "Success",
			msg:         "Running command",
			cmd:         "echo",
			args:        []string{"hello"},
			expectedErr: nil,
		},
		{
			name:        "Error",
			msg:         "Running command",
			cmd:         "nonexistentcommand",
			args:        []string{"arg1", "arg2"},
			expectedErr: exec.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmdErr := RunCmd(tc.msg, tc.cmd, tc.args...)
			if cmdErr != tc.expectedErr {
				t.Errorf("Expected error: %v, got: %v", tc.expectedErr, cmdErr)
			}
			if cmdErr == nil {
				if !strings.Contains(buf.String(), strings.Join(tc.args, " ")) {
					t.Errorf("Expected output to contain command arguments: %v", tc.args)
				}
			}
		})
	}
}
