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

	if len(os.Args) > 1 {
		role := os.Args[1]

		config, err := loadConfig()
		must(err)

		roleConfig, ok := config[role]
		if !ok {
			must(fmt.Errorf("%s not in ~/.aws/roles", role))
		}

		creds, err := assumeRole(roleConfig.IAMProfile, roleConfig.Role, roleConfig.MFA)
		must(err)

		err = saveCredentials(role, roleConfig.TMPProfile, roleConfig.Region, creds)
		must(err)

	} else {
		usage()
		os.Exit(1)
	}
}

func setProfile(profile string) error {

	os.Setenv("AWS_PROFILE", profile)

	return syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
}

func saveSession(session, role string) error {

	sessionFilePath := fmt.Sprintf("%s/.aws/session.%s", os.Getenv("HOME"), role)

	current := session

	//timestamp := current.Unix()

	/*
				int32(time.Now().Unix())

				now := time.Now()
		    secs := now.Unix()
		    nanos := now.UnixNano()
		    fmt.Println(now)
	*/

	SaveCmd := exec.Command("echo", session, ">", sessionFilePath)
	if err := SaveCmd.Start(); err != nil {
		return err
	}

	if err := SaveCmd.Wait(); err != nil {
		return err
	}

	fmt.Println(current)

	return nil
}

type credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      string
}

func saveCredentials(role, profile, region string, creds *credentials) error {

	// Profile
	profileFormat := fmt.Sprintf("profile.%s.region", profile)
	profileArgs := []string{"configure", "set", profileFormat, region}
	profileCmd := exec.Command("aws", profileArgs...)
	if err := profileCmd.Start(); err != nil {
		return err
	}

	if err := profileCmd.Wait(); err != nil {
		return err
	}

	// Access Key ID
	accessKeyIDFormat := fmt.Sprintf("profile.%s.aws_access_key_id", profile)
	accessKeyIDArgs := []string{"configure", "set", accessKeyIDFormat, creds.AccessKeyID}
	accessCmd := exec.Command("aws", accessKeyIDArgs...)
	if err := accessCmd.Start(); err != nil {
		return err
	}

	if err := accessCmd.Wait(); err != nil {
		return err
	}

	// Secret Access Key
	secretAccessKeyFormat := fmt.Sprintf("profile.%s.aws_secret_access_key", profile)
	secretAccessKeyArgs := []string{"configure", "set", secretAccessKeyFormat, creds.SecretAccessKey}
	secretAccessCmd := exec.Command("aws", secretAccessKeyArgs...)
	if err := secretAccessCmd.Start(); err != nil {
		return err
	}

	if err := secretAccessCmd.Wait(); err != nil {
		return err
	}

	// Session Token
	sessionTokenFormat := fmt.Sprintf("profile.%s.aws_session_token", profile)
	sessionTokenArgs := []string{"configure", "set", sessionTokenFormat, creds.SessionToken}
	sessionTokenCmd := exec.Command("aws", sessionTokenArgs...)
	if err := sessionTokenCmd.Start(); err != nil {
		return err
	}

	if err := sessionTokenCmd.Wait(); err != nil {
		return err
	}

	expiration := string(creds.Expiration)

	fmt.Println(expiration)

	saveSession(expiration, role)
	setProfile(profile)

	return nil

}

func assumeRole(iamProfile, role, mfa string) (*credentials, error) {
	args := []string{
		"--debug",
		"sts",
		"assume-role",
		"--profile", iamProfile,
		"--output", "json",
		"--role-arn", role,
		"--role-session-name", "cli",
		"--query", "[Credentials.AccessKeyId,Credentials.SecretAccessKey,Credentials.SessionToken,Credentials.Expiration]",
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
	IAMProfile string `yaml:"iam_profile"`
	TMPProfile string `yaml:"tmp_profile"`
	Region     string `yaml:"region"`
	Role       string `yaml:"role"`
	MFA        string `yaml:"mfa"`
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
