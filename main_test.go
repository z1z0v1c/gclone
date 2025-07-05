package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestCommandExecution tests that commands run and return correct output
func TestCommandExecution(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping processes isolation test: requires root privileges")
	}

	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  bool
	}{
		{
			name:           "echo command",
			args:           []string{"run", "alpine", "echo", "Hello World"},
			expectedOutput: "Hello World\n",
			expectedError:  false,
		},
		{
			name:           "pwd command",
			args:           []string{"run", "alpine", "pwd"},
			expectedOutput: "/\n",
			expectedError:  false,
		},
		{
			name:          "invalid command",
			args:          []string{"run", "alpine", "noncommand"},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", append([]string{"run", "main.go"}, tt.args...)...)
			output, err := cmd.CombinedOutput()

			if tt.expectedError {
				if err == nil {
					t.Errorf("No expected error.")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %q.", err.Error())
				return
			}

			if string(output) != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, string(output))
			}
		})
	}
}

// TestExitCodes verifies that exit codes are properly propagated
func TestExitCodes(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping processes isolation test: requires root privileges")
	}

	tests := []struct {
		name         string
		args         []string
		expectedCode int
	}{
		{
			name:         "successful command",
			args:         []string{"run", "alpine", "true"},
			expectedCode: 0,
		},
		{
			name:         "failing command",
			args:         []string{"run", "alpine", "false"},
			expectedCode: 1,
		},
		{
			name:         "ls nonexistent file",
			args:         []string{"run", "alpine", "ls", "nonexistent_file"},
			expectedCode: 1, // TODO "No such file or directory" should return 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Needs a compiled binary for proper error propagation
			cmd := exec.Command("./gocker", tt.args...)
			err := cmd.Run()

			var exitCode int
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("Unexpected error type: %v", err)
				}
			}

			if exitCode != tt.expectedCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectedCode, exitCode)
			}
		})
	}
}

// TestNamespaceIsolation tests that UTS namespace is properly isolated
func TestNamespaceIsolation(t *testing.T) {
	// Skip test if not running as root (required for namespaces)
	if os.Geteuid() != 0 {
		t.Skip("Skipping namespace test (requires root privileges)")
	}

	// Get original hostname
	originalHostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("Failed to get original hostname: %v", err)
	}

	// Test that container has different hostname
	cmd := exec.Command("go", "run", "main.go", "run", "alpine", "hostname")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run container: %v", err)
	}

	containerHostname := strings.TrimSpace(string(output))
	if containerHostname != "container" {
		t.Errorf("Expected container hostname 'container', got %q", containerHostname)
	}

	// Verify host hostname unchanged
	currentHostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("Failed to get current hostname: %v", err)
	}

	if currentHostname != originalHostname {
		t.Errorf("Host hostname changed from %q to %q", originalHostname, currentHostname)
	}
}

// TestFilesystemIsolation tests that the container can't access host files
func TestFileSystemIsolation(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping filesystem isolation test: requires root privileges")
	}

	// Create a unique test file on the host
	hostTestFile := "/tmp/host_test_file_" + fmt.Sprintf("%d", time.Now().UnixNano())
	if err := os.WriteFile(hostTestFile, []byte("host content"), 0644); err != nil {
		t.Fatalf("Failed to create host test file: %v", err)
	}
	defer os.Remove(hostTestFile)

	// Try to access the host file from within the container
	cmd := exec.Command("./gocker", "run", "alpine", "/bin/busybox", "cat", hostTestFile)

	// The command should fail because the file shouldn't be accessible
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("Container was able to access host file %s, output: %s", hostTestFile, string(output))
	}

	// The error should indicate file not found
	if !strings.Contains(string(output), "No such file") {
		t.Logf("Expected 'No such file' error, got: %s", string(output))
	}
}

// TestContainerRootFilesystem tests that container sees Alpine filesystem as root
func TestContainerRootFileSystem(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping container root filesystem test: requires root privileges")
	}

	// List root directory contents
	cmd := exec.Command("./gocker", "run", "alpine", "/bin/busybox", "ls", "/")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list container root directory: %v", err)
	}

	outputStr := string(output)

	// Check for standard files/directories
	expectedItems := []string{"bin", "etc", "lib", "usr", "var"}
	for _, item := range expectedItems {
		if !strings.Contains(outputStr, item) {
			t.Errorf("Expected to find %s in container root, got: %s", item, outputStr)
		}
	}

	// Check that we don't see host-specific directories
	hostSpecificItems := []string{"boot", "lib64"}
	for _, item := range hostSpecificItems {
		if strings.Contains(outputStr, item) {
			t.Logf("Warning: Found host-specific item %s in container root: %s", item, outputStr)
		}
	}

	t.Logf("Container root directory contents: %s", outputStr)
}

// TestChrootPreventsEscape tests that cd .. doesn't escape the container root
func TestChrootPreventsEscape(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping chroot escape test: requires root privileges")
	}

	script := `
		pwd
		cd ..
		pwd
		cd ../../..
		pwd
	`

	cmd := exec.Command("./gocker", "run", "alpine", "/bin/busybox", "sh", "-c", script)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run chroot escape test: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// All pwd commands should return '/'
	for i, line := range lines {
		if line != "/" {
			t.Errorf("Line %d: expected '/', got '%s'. Full output: %s", i+1, line, string(output))
		}
	}

	t.Logf("Chroot escape test output: %s", string(output))
}

// TestFilesystemWriteIsolation tests that writes in container don't affect host
func TestFilesystemWriteIsolation(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping filesystem write isolation test: requires root privileges")
	}

	containerTestFile := "/tmp/container_test_file"

	// Create a test file in the container
	cmd := exec.Command("./gocker", "run", "alpine", "/bin/busybox", "sh", "-c",
		fmt.Sprintf("touch %s", containerTestFile))

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create file in container: %v", err)
	}

	// Write to the test file
	testContent := "container test content"
	cmd = exec.Command("./gocker", "run", "alpine", "/bin/busybox", "sh", "-c",
		fmt.Sprintf("echo '%s' > %s", testContent, containerTestFile))

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to write file in container: %v", err)
	}

	// Verify the file exists in the container
	cmd = exec.Command("./gocker", "run", "alpine", "/bin/busybox", "cat", containerTestFile)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to read file from container: %v", err)
	}

	if !strings.Contains(string(output), testContent) {
		t.Errorf("Container file content mismatch. Expected: %s, Got: %s", testContent, string(output))
	}

	// Verify the file doesn't exist on the host at /tmp/container_test_file
	if _, err := os.Stat(containerTestFile); err == nil {
		t.Error("Container file leaked to host filesystem")
	}
}

// TestProcessesIsolation tests that processes are properly isolated
func TestProcessesIsolation(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping processes isolation test: requires root privileges")
	}

	cmd := exec.Command("./gocker", "run", "alpine", "ps", "axf")

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run ps in container: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Count processes
	processCount := 0
	for i, line := range lines {
		if i == 0 {
			continue // Skip header
		}
		if strings.TrimSpace(line) != "" {
			processCount++
		}
	}

	// Should have very few processes (container init + command + maybe children)
	if processCount > 5 {
		t.Errorf("Too many processes in container (%d), isolation may not be working. Output:\n%s",
			processCount, output)
	}
}

// TestUserNamespaceIsolation tests that the container runs without root privileges on the host
func TestUserNamespaceIsolation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping User namespace isolation test: requires non-root privileges")
	}

	// Get current host user info
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	hostUID, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		t.Fatalf("Failed to parse host UID: %v", err)
	}

	hostGID, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		t.Fatalf("Failed to parse host GID: %v", err)
	}

	t.Logf("Host user: %s (UID: %d, GID: %d)", currentUser.Username, hostUID, hostGID)

	// Start a long-running process in the container
	cmd := exec.Command("./gocker", "run", "alpine", "sleep", "30")

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Give the container time to start
	time.Sleep(2 * time.Second)

	// Find the sleep process on the host
	psCmd := exec.Command("ps", "-eo", "pid,uid,gid,user,command")
	output, err := psCmd.Output()
	if err != nil {
		t.Fatalf("Failed to run ps command: %v", err)
	}

	// Parse ps output to find our sleep process
	lines := strings.Split(string(output), "\n")
	var sleepPID int
	var sleepUID, sleepGID int
	var sleepUser string

	for _, line := range lines {
		if strings.Contains(line, "sleep 30") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				sleepPID, _ = strconv.Atoi(fields[0])
				sleepUID, _ = strconv.Atoi(fields[1])
				sleepGID, _ = strconv.Atoi(fields[2])
				sleepUser = fields[3]
				break
			}
		}
	}

	// Clean up the container process
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	if sleepPID == 0 {
		t.Fatal("Could not find sleep process in host process list")
	}

	t.Logf("Container sleep process: PID=%d, UID=%d, GID=%d, User=%s", sleepPID, sleepUID, sleepGID, sleepUser)

	// Test 1: The process should NOT be running as root (UID 0) on the host
	if sleepUID == 0 {
		t.Error("FAIL: Container process is running as root (UID 0) on the host - user namespace isolation not working")
	} else {
		t.Logf("PASS: Container process is not running as root on the host (UID: %d)", sleepUID)
	}

	// Test 2: The process should be running as the host user (user namespace mapping)
	if sleepUID != hostUID {
		t.Errorf("FAIL: Expected container process to run as host user UID %d, but got UID %d", hostUID, sleepUID)
	} else {
		t.Logf("PASS: Container process is running as expected host user (UID: %d)", sleepUID)
	}

	// Test 3: The process should be running with the host user's GID
	if sleepGID != hostGID {
		t.Errorf("FAIL: Expected container process to run with host GID %d, but got GID %d", hostGID, sleepGID)
	} else {
		t.Logf("PASS: Container process is running with expected host GID (%d)", sleepGID)
	}

	// Test 4: The username should match the host user
	if sleepUser != currentUser.Username {
		t.Errorf("FAIL: Expected container process to run as user '%s', but got '%s'", currentUser.Username, sleepUser)
	} else {
		t.Logf("PASS: Container process is running as expected user (%s)", sleepUser)
	}
}

func TestMemoryLimit(t *testing.T) {
	cmd := exec.Command("./gocker", "run", "alpine", "sh", "-c", "dd if=/dev/zero of=/dev/null bs=1M count=1000")
	err := cmd.Run()
	if err == nil {
		t.Error("Expected memory limit to kill the process, but it ran successfully")
	}
}

func TestCPULimit(t *testing.T) {
	start := time.Now()

	cmd := exec.Command("./gocker", "run", "alpine", "sh", "-c", `
		i=0; while [ $i -lt 100000 ]; do :; i=$((i+1)); done
	`)
	err := cmd.Run()

	duration := time.Since(start)

	if err != nil {
		t.Fatalf("CPU-bound task failed: %v", err)
	}

	t.Logf("CPU-bound task duration: %v", duration)

	// Expect this to take longer if CPU limit is enforced
	if duration < 2*time.Second {
		t.Error("CPU limit may not be enforced: task finished too quickly")
	}
}
