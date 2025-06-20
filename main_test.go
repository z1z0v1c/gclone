package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCommandExecution(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  bool
	}{
		{
			name:           "echo command",
			args:           []string{"run", "echo", "Hello World"},
			expectedOutput: "Hello World\n",
			expectedError:  false,
		},
		{
			name:           "pwd command",
			args:           []string{"run", "pwd"},
			expectedOutput: "gocker\n",
			expectedError:  false,
		},
		{
			name:          "invalid command",
			args:          []string{"run", "noncommand"},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", append([]string{"run", "main.go"}, tt.args...)...)
			output, err := cmd.CombinedOutput()

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)

				return
			}

			// Check suffix for pwd command (also enough for echo)
			if !strings.HasSuffix(string(output), tt.expectedOutput) {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, string(output))
			}
		})
	}
}
