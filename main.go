package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"gopkg.in/yaml.v2"
)

var configFilePath = fmt.Sprintf("%s/.aws/roles", os.Getenv("HOME"))

func usage() {
	fmt.Println("Usage: assume-role <role>")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	role := os.Args[1]

	config, err := loadConfig()
	must(err)

	roleConfig, ok := config[role]
	if !ok {
		must(fmt.Errorf("%s not in ~/.aws/roles", role))
	}

	creds, err := assumeRole(roleConfig.Profile, roleConfig.Role, roleConfig.MFA)
	must(err)

	err = saveCredentials(role, roleConfig.Region, creds)
	must(err)
}

func setProfile(role string) error {

	profile := fmt.Sprintf("ho-mfa-%s", role)
	os.Setenv("AWS_PROFILE", profile)

	return syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
}

type credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

func saveCredentials(role, region string, creds *credentials) error {

	fmt.Printf("export AWS_PROFILE=ho-mfa-%s\n", role)

	// Profile
	profileFormat := fmt.Sprintf("profile.ho-mfa-%s.region", role)
	profileArgs := []string{"configure", "set", profileFormat, region}
	profileCmd := exec.Command("aws", profileArgs...)
	if err := profileCmd.Start(); err != nil {
		return err
	}

	if err := profileCmd.Wait(); err != nil {
		return err
	}

	// Access Key ID
	accessKeyIDFormat := fmt.Sprintf("profile.ho-mfa-%s.aws_access_key_id", role)
	accessKeyIDArgs := []string{"configure", "set", accessKeyIDFormat, creds.AccessKeyID}
	accessCmd := exec.Command("aws", accessKeyIDArgs...)
	if err := accessCmd.Start(); err != nil {
		return err
	}

	if err := accessCmd.Wait(); err != nil {
		return err
	}

	// Secret Access Key
	secretAccessKeyFormat := fmt.Sprintf("profile.ho-mfa-%s.aws_secret_access_key", role)
	secretAccessKeyArgs := []string{"configure", "set", secretAccessKeyFormat, creds.SecretAccessKey}
	secretAccessCmd := exec.Command("aws", secretAccessKeyArgs...)
	if err := secretAccessCmd.Start(); err != nil {
		return err
	}

	if err := secretAccessCmd.Wait(); err != nil {
		return err
	}

	// Session Token
	sessionTokenFormat := fmt.Sprintf("profile.ho-mfa-%s.aws_session_token", role)
	sessionTokenArgs := []string{"configure", "set", sessionTokenFormat, creds.SessionToken}
	sessionTokenCmd := exec.Command("aws", sessionTokenArgs...)
	if err := sessionTokenCmd.Start(); err != nil {
		return err
	}

	if err := sessionTokenCmd.Wait(); err != nil {
		return err
	}

	setProfile(role)

	return nil

}

func assumeRole(profile, role, mfa string) (*credentials, error) {
	args := []string{
		"sts",
		"assume-role",
		"--profile", profile,
		"--output", "json",
		"--role-arn", role,
		"--role-session-name", "cli",
	}
	if mfa != "" {
		args = append(args,
			"--serial-number", mfa,
			"--token-code",
			readTokenCode(),
		)
	}

	b := new(bytes.Buffer)
	cmd := exec.Command("aws", args...)
	cmd.Stdout = b
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	var resp struct{ Credentials credentials }
	if err := json.NewDecoder(b).Decode(&resp); err != nil {
		return nil, err
	}

	return &resp.Credentials, nil
}

type roleConfig struct {
	Profile string `yaml:"profile"`
	Region  string `yaml:"region"`
	Role    string `yaml:"role"`
	MFA     string `yaml:"mfa"`
}

type config map[string]roleConfig

func readTokenCode() string {
	r := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stderr, "MFA code: ")
	text, _ := r.ReadString('\n')
	return strings.TrimSpace(text)
}

func loadConfig() (config, error) {
	raw, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	config := make(config)
	return config, yaml.Unmarshal(raw, &config)
}

func must(err error) {
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
