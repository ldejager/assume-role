package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

var configFilePath = fmt.Sprintf("%s/.aws/roles", os.Getenv("HOME"))
var awsProfile string

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

		cleanEnv()

		if len(roleConfig.TMPProfile) > 0 {
			awsProfile = roleConfig.TMPProfile
		} else {
			awsProfile = role
		}

		checkSession(role)

		creds, err := assumeRole(roleConfig.IAMProfile, roleConfig.Role, roleConfig.MFA)
		must(err)

		err = saveCredentials(role, awsProfile, roleConfig.Region, creds)
		must(err)

	} else {
		usage()
		os.Exit(1)
	}
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

func cleanEnv() {
	os.Unsetenv("AWS_PROFILE")
}

func setProfile(profile string) error {

	os.Setenv("AWS_PROFILE", profile)

	return syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
}

func saveSession(session, role string) error {

	sessionFilePath := fmt.Sprintf("%s/.aws/session.new.%s", os.Getenv("HOME"), role)

	t, err := time.Parse(time.RFC3339Nano, session)
	timestamp := t.Unix()

	fileHandle, err := os.Create(sessionFilePath)
	checkErr(err)
	writer := bufio.NewWriter(fileHandle)
	defer fileHandle.Close()
	fmt.Fprintln(writer, timestamp)
	writer.Flush()

	return nil
}

func sessionRemaining(a, b int64) string {
	return strconv.FormatInt(a-b, 10)
}

func getSession(role string) int64 {

	sessionFilePath := fmt.Sprintf("%s/.aws/session.new.%s", os.Getenv("HOME"), role)

	getTimestamp, err := ioutil.ReadFile(sessionFilePath)
	checkErr(err)

	timestamp := int64(len(getTimestamp))

	return timestamp
}

func checkSession(role string) string {

	session := getSession(role)
	current := time.Now().Unix()

	validatedSession := sessionRemaining(session, current)

	if validatedSession < "0" {
		return fmt.Sprintf("Session expired, please revalidate")
	} else {
		fmt.Sprintf("Session valid")
		os.Exit(1)
		return string("1")
	}
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

	saveSession(creds.Expiration, role)
	setProfile(profile)

	return nil

}

func assumeRole(iamProfile, role, mfa string) (*credentials, error) {

	args := []string{
		"sts",
		"assume-role",
		"--profile", iamProfile,
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
