package test

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimple(t *testing.T) {
	// Removed t.Parallel() due to Docker container name conflicts

	// Copy the specific test subdirectory
	sourceDir := filepath.Join("test", "simple")
	testDir := test_structure.CopyTerraformFolderToTemp(t, "../", sourceDir)

	// Generate SSH keys first
	var sshKeys *testSSHKeys
	test_structure.RunTestStage(t, "generate_ssh_keys", func() {
		sshKeys = generateSSHKeys(t, testDir)
	})

	test_structure.RunTestStage(t, "setup_docker", func() {
		setupDokkuContainer(t)
	})

	test_structure.RunTestStage(t, "setup_ssh", func() {
		setupSSH(t, sshKeys)
	})

	test_structure.RunTestStage(t, "apply_terraform", func() {
		// Define test-specific configuration inline
		appName := "simple-test-app"

		// Base terraform options
		vars := map[string]interface{}{
			"dokku_host":      "localhost",
			"dokku_port":      3022,
			"ssh_private_key": sshKeys.privateKeyPEM,
			"app_name":        appName,
		}

		terraformOptions := &terraform.Options{
			TerraformDir: testDir,
			Vars:         vars,
			NoColor:      true,
		}

		// Build the provider first
		buildProvider(t, testDir)

		// Generate terraform RC for dev overrides
		generateTerraformRC(t, testDir)

		// Apply directly since we have provider development overrides
		terraform.Apply(t, terraformOptions)

		// Store options for cleanup
		test_structure.SaveTerraformOptions(t, testDir, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate_dokku", func() {
		appName := "simple-test-app"

		keyPair := &ssh.KeyPair{
			PublicKey:  sshKeys.publicKeySSH,
			PrivateKey: sshKeys.privateKeyPEM,
		}

		host := ssh.Host{
			Hostname:    "localhost",
			SshKeyPair:  keyPair,
			SshUserName: "dokku",
			CustomPort:  3022,
		}

		validateSimpleApp(t, host, appName)
	})

	test_structure.RunTestStage(t, "validate_http", func() {
		// Test HTTP accessibility
		maxRetries := 10
		retryInterval := 3 * time.Second

		var httpResponse string
		var httpErr error

		for i := 0; i < maxRetries; i++ {
			// Use curl to test HTTP connectivity since the app should be accessible via the exposed port
			curlCmd := exec.Command("curl", "-s", "-f", "http://localhost:8080")
			output, err := curlCmd.Output()

			if err == nil {
				httpResponse = string(output)
				httpErr = nil
				break
			}

			httpErr = err
			t.Logf("HTTP test attempt %d failed: %v", i+1, err)
			if i < maxRetries-1 {
				time.Sleep(retryInterval)
			}
		}

		require.NoError(t, httpErr, "Failed to get HTTP response from app after %d retries", maxRetries)

		// Verify the response contains expected content from jmalloc/echo-server
		assert.Contains(t, httpResponse, "Request served by", "HTTP response should contain 'Request served by' indicating the echo-server container is serving content")
		assert.Contains(t, httpResponse, "Host: localhost:8080", "HTTP response should show the app received the request on port 8080")

		t.Logf("HTTP validation successful! Response preview: %.200s...", httpResponse)
	})

	test_structure.RunTestStage(t, "destroy_terraform", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testDir)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "cleanup_docker", func() {
		cleanupDocker(t)
	})
}
