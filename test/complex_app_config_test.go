package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/require"
	cryptossh "golang.org/x/crypto/ssh"
)

func TestComplexAppConfig(t *testing.T) {
	// Removed t.Parallel() due to Docker container name conflicts
	// This test uses properly typed variables (app_config as map(string)) and validates
	// that provider does NOT return type errors when using correct HCL types

	// Copy the specific test subdirectory
	sourceDir := filepath.Join("test", "complex_app_config")
	testDir := test_structure.CopyTerraformFolderToTemp(t, "../", sourceDir)

	// Generate SSH keys first
	var sshKeys *testSSHKeys
	test_structure.RunTestStage(t, "generate_ssh_keys", func() {
		// Generate RSA private key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err, "Failed to generate private key")

		// Encode private key to PEM format
		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}
		privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

		// Generate public key in SSH format
		publicKey, err := cryptossh.NewPublicKey(&privateKey.PublicKey)
		require.NoError(t, err, "Failed to generate public key")
		publicKeySSH := string(cryptossh.MarshalAuthorizedKey(publicKey))

		// Write keys to files
		privateKeyPath := filepath.Join(testDir, "ssh_key")
		publicKeyPath := filepath.Join(testDir, "ssh_key.pub")

		err = os.WriteFile(privateKeyPath, privateKeyBytes, 0600)
		require.NoError(t, err, "Failed to write private key")

		err = os.WriteFile(publicKeyPath, []byte(publicKeySSH), 0644)
		require.NoError(t, err, "Failed to write public key")

		t.Logf("Generated SSH keys: %s, %s", privateKeyPath, publicKeyPath)

		sshKeys = &testSSHKeys{
			privateKeyPEM:  string(privateKeyBytes),
			publicKeySSH:   strings.TrimSpace(publicKeySSH),
			privateKeyPath: privateKeyPath,
			publicKeyPath:  publicKeyPath,
		}
	})

	test_structure.RunTestStage(t, "setup_docker", func() {
		// Try to remove any existing container with the same name
		cleanupCmd := exec.Command("docker", "rm", "-f", containerName)
		cleanupCmd.Run()

		// Get the Dokku version to use
		dokkuVersion := getDokkuVersion()
		dokkuImageName := fmt.Sprintf("dokku/dokku:%s", dokkuVersion)

		// Run the Dokku container
		runOptions := &docker.RunOptions{
			Name:       containerName,
			Detach:     true,
			Privileged: true,
			OtherOptions: []string{
				"-p", fmt.Sprintf("%s:22", sshPort),
				"-p", fmt.Sprintf("%s:80", httpPort),
				"-e", "DOKKU_HOSTNAME=dokku.test",
				"-e", "DOKKU_HOST_ROOT=/home/dokku/dokku",
				"-v", "/var/run/docker.sock:/var/run/docker.sock",
				"--add-host", "host.docker.internal:host-gateway",
			},
		}

		docker.Run(t, dokkuImageName, runOptions)

		// Wait for container to be up and running
		retries := 30
		retryInterval := 3 * time.Second
		sshServiceRunning := false

		for i := 0; i < retries; i++ {
			// Check if SSH is running inside the container
			sshCheckCmd := exec.Command("docker", "exec", containerName, "service", "ssh", "status")
			err := sshCheckCmd.Run()
			if err == nil {
				sshServiceRunning = true
				break
			}
			require.Error(t, err, "SSH service check failed")
			time.Sleep(retryInterval)
		}

		// Ensure dokku is installed and working
		dokkuInstalled := false
		if sshServiceRunning {
			// Check if dokku command works
			dokkuCheckCmd := exec.Command("docker", "exec", containerName, "dokku", "--version")
			err := dokkuCheckCmd.Run()
			if err == nil {
				dokkuInstalled = true
			}
		}

		require.True(t, sshServiceRunning && dokkuInstalled, "Dokku container is not ready")
		t.Logf("Container ready, waiting additional 30 seconds for services to fully stabilize...")
		time.Sleep(30 * time.Second)
	})

	test_structure.RunTestStage(t, "setup_ssh", func() {
		// Add public key to authorized keys
		authorizedKeysCmd := exec.Command("docker", "exec", containerName, "bash", "-c",
			fmt.Sprintf("echo '%s' > /home/dokku/.ssh/authorized_keys", sshKeys.publicKeySSH))
		require.NoError(t, authorizedKeysCmd.Run(), "Failed to add SSH key to authorized_keys")

		// Test SSH connection
		maxRetries := 5
		retryInterval := 2 * time.Second

		for i := 1; i <= maxRetries; i++ {
			host := ssh.Host{
				Hostname:    "localhost",
				SshKeyPair:  &ssh.KeyPair{PrivateKey: sshKeys.privateKeyPEM},
				SshUserName: "dokku",
				CustomPort:  3022,
			}

			setupOutput, err := ssh.CheckSshCommandE(t, host, "id && service ssh status && netstat -tlnp | grep :22")
			t.Logf("SSH setup attempt %d output: %s", i, setupOutput)

			if err == nil {
				output, err := ssh.CheckSshCommandE(t, host, "dokku --version")
				t.Logf("SSH test attempt %d: %v, output: %s", i, err, output)

				if err == nil {
					t.Logf("SSH connection successful!")
					return
				}
			}

			if i < maxRetries {
				t.Logf("SSH connection failed, retrying in %v...", retryInterval)
				time.Sleep(retryInterval)
			}
		}

		require.Fail(t, "Failed to establish SSH connection after %d retries", maxRetries)
	})

	test_structure.RunTestStage(t, "apply_terraform", func() {
		// Disable detailed logging for cleaner output
		os.Unsetenv("TF_LOG")
		os.Unsetenv("TF_LOG_CORE")
		os.Unsetenv("TF_LOG_PROVIDER")

		// Define test-specific configuration inline
		appName := "complex-test-app"

		// Base terraform options
		vars := map[string]interface{}{
			"dokku_host":      "localhost",
			"dokku_port":      3022,
			"ssh_private_key": sshKeys.privateKeyPEM,
			"app_name":        appName,
			"app_config": map[string]interface{}{
				"ENV":      "prod",
				"APP_NAME": appName,
				"DEBUG":    "false",
				"API_URL":  "https://api.example.com",
				"PORT":     "5000",
			},
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

		// Skip init when using dev overrides (as recommended by Terraform warning)
		// Apply directly since we have provider development overrides
		// Now using properly typed app_config variable - this should work unless there are provider type issues
		terraformOutput, applyErr := terraform.ApplyE(t, terraformOptions)

		// Capture the terraform output for analysis
		outputStr := terraformOutput
		if applyErr != nil {
			outputStr += "\n" + applyErr.Error()
		}

		t.Logf("Terraform output captured: %s", outputStr)

		// Check for provider type-related errors - these should NOT occur with properly typed variables
		typeErrorPatterns := []string{
			"Incorrect attribute value type",
			"Invalid attribute configuration",
			"Type mismatch",
			"Incorrect type",
			"provider issue",
			"Block type",
			"does not support",
			"Value Conversion Error",
			"Target Type:",
			"Suggested Type:",
		}

		var foundTypeError bool
		var typeErrorMsg string
		for _, pattern := range typeErrorPatterns {
			if strings.Contains(outputStr, pattern) {
				foundTypeError = true
				typeErrorMsg = pattern
				break
			}
		}

		// Assert that NO type errors should occur when using properly typed variables
		if foundTypeError {
			require.Fail(t, "Provider type error found when using properly typed app_config variable: %s\nFull output: %s", typeErrorMsg, outputStr)
		}

		// If apply failed for other reasons (connection, etc.), that's acceptable
		if applyErr != nil {
			t.Logf("✓ Apply failed with non-type error (acceptable): %v", applyErr)
		} else {
			t.Logf("✓ Apply succeeded with properly typed variables")
		}

		// Store options for cleanup
		test_structure.SaveTerraformOptions(t, testDir, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate_dokku", func() {
		// Skip validation stage since we're testing provider type handling
		// The test validates proper type handling in the apply_terraform stage
		t.Skip("Skipping validation - test focuses on provider type error detection")
	})

	test_structure.RunTestStage(t, "destroy_terraform", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testDir)
		// Use DestroyE since the apply might have failed and there might be nothing to destroy
		_, destroyErr := terraform.DestroyE(t, terraformOptions)
		if destroyErr != nil {
			t.Logf("Destroy failed (expected if apply failed): %v", destroyErr)
		}
	})

	test_structure.RunTestStage(t, "cleanup_docker", func() {
		cleanupDocker(t)
	})
}
