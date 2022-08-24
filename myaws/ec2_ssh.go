package myaws

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal" // nolint: staticcheck
)

// EC2SSHOptions customize the behavior of the SSH command.
type EC2SSHOptions struct {
	FilterTag    string
	LoginName    string
	IdentityFile string
	Private      bool
	Command      string
}

// EC2SSH resolves IP address of EC2 instance and connects to it by SSH.
func (client *Client) EC2SSH(options EC2SSHOptions) error {
	config, err := buildSSHConfig(options.LoginName, options.IdentityFile)
	if err != nil {
		return err
	}

	instances, err := client.FindEC2Instances(options.FilterTag, false)
	if err != nil {
		return err
	}

	if len(instances) == 0 {
		return errors.Errorf("no such instance: %s", options.FilterTag)
	}

	if len(instances) >= 2 && options.Command == "" {
		return errors.Errorf("multiple instances found")
	}

	hostnames := []string{}
	for _, instance := range instances {
		hostname, err := client.resolveEC2IPAddress(instance, options.Private)
		if err != nil {
			return err
		}
		hostnames = append(hostnames, hostname)
	}

	// Start single ssh session with terminal
	if options.Command == "" {
		return client.startSSHSessionWithTerminal(hostnames[0], "22", config)
	}

	// Execute ssh command to multiple hosts in series
	for _, hostname := range hostnames {
		if err := client.executeSSHCommand(hostname, "22", config, options.Command); err != nil {
			return err
		}
	}

	return nil
}

func (client *Client) resolveEC2IPAddress(instance *ec2.Instance, private bool) (string, error) {
	if private {
		return client.resolveEC2PrivateIPAddress(instance)
	}
	return client.resolveEC2PublicIPAddress(instance)
}

func (client *Client) resolveEC2PrivateIPAddress(instance *ec2.Instance) (string, error) {
	if instance.PrivateIpAddress == nil {
		return "", errors.Errorf("no private ip address: %s", *instance.InstanceId)
	}
	return *instance.PrivateIpAddress, nil
}

func (client *Client) resolveEC2PublicIPAddress(instance *ec2.Instance) (string, error) {
	if instance.PublicIpAddress == nil {
		return "", errors.Errorf("no public ip address: %s", *instance.InstanceId)
	}
	return *instance.PublicIpAddress, nil
}

func buildSSHConfig(loginName string, identityFile string) (*ssh.ClientConfig, error) {
	normalizedIdentityFile := strings.Replace(identityFile, "~", os.Getenv("HOME"), 1)
	key, err := os.ReadFile(normalizedIdentityFile)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read private key:")
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse private key:")
	}

	config := &ssh.ClientConfig{
		User: loginName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // nolint: gosec
	}

	return config, nil
}

func buildSSHSessionPipe(session *ssh.Session) error {
	stdin, err := session.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "unable to setup stdin for session:")
	}
	go io.Copy(stdin, os.Stdin) // nolint: errcheck

	stdout, err := session.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "unable to setup stdout for session:")
	}
	go io.Copy(os.Stdout, stdout) // nolint: errcheck

	stderr, err := session.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "unable to setup stderr for session:")
	}
	go io.Copy(os.Stderr, stderr) // nolint: errcheck

	return nil
}

func (client *Client) startSSHSessionWithTerminal(hostname string, port string, config *ssh.ClientConfig) error {
	addr := fmt.Sprintf("%s:%s", hostname, port)
	connection, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return errors.Wrap(err, "unable to connect:")
	}
	defer connection.Close()

	session, err := connection.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to new session failed:")
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		return errors.Wrap(err, "unable to put terminal in Raw Mode:")
	}
	defer terminal.Restore(fd, oldState) // nolint: errcheck

	width, height, _ := terminal.GetSize(fd)

	if err := session.RequestPty("xterm", height, width, modes); err != nil {
		return errors.Wrap(err, "request for pseudo terminal failed:")
	}

	if err := buildSSHSessionPipe(session); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		return errors.Wrap(err, "failed to start shell:")
	}
	session.Wait() // nolint: errcheck

	return nil
}

func (client *Client) executeSSHCommand(hostname string, port string, config *ssh.ClientConfig, command string) error {
	addr := fmt.Sprintf("%s:%s", hostname, port)
	connection, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return errors.Wrap(err, "unable to connect:")
	}
	defer connection.Close()

	session, err := connection.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to new session failed:")
	}
	defer session.Close()

	// Request pty for sudo
	fd := int(os.Stdin.Fd())
	width, height, _ := terminal.GetSize(fd)
	if err := session.RequestPty("xterm", height, width, ssh.TerminalModes{}); err != nil {
		return errors.Wrap(err, "request for pseudo terminal failed:")
	}

	out, err := session.CombinedOutput(command)

	fmt.Fprintf(client.stdout, "========== Start output on host: %s ==========\n", hostname)
	fmt.Fprintln(client.stdout, string(out))
	fmt.Fprintf(client.stdout, "========== End   output on host: %s ==========\n", hostname)

	if err != nil {
		return errors.Wrapf(err, "failed to execute command: %s", command)
	}

	return nil
}
