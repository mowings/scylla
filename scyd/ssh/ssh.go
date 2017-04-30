package ssh

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

type Keychain struct {
	signers []ssh.Signer
}

type SshConnection struct {
	config          *ssh.ClientConfig
	original_server string
	timeout         int
	server          string
	network_conn    net.Conn
	client_conn     ssh.Conn
	client          *ssh.Client
	SudoCommand     string
}

const NO_TIMEOUT = 0

var shellescape_re = regexp.MustCompile("([^A-Za-z0-9_\\-.,:\\/@\n])")

func Shellescape(str string) string {
	return shellescape_re.ReplaceAllString(str, "\\$1")
}

func (k *Keychain) Key(i int) (key ssh.PublicKey, err error) {
	if i >= len(k.signers) {
		return nil, nil
	}
	return k.signers[i].PublicKey(), nil
}

func (k *Keychain) Sign(i int, rand io.Reader, data []byte) (signature *ssh.Signature, err error) {
	if i >= len(k.signers) {
		return
	}
	signature, err = k.signers[i].Sign(rand, data)
	return
}

func mkSigner(key_filename string) (signer ssh.Signer, err error) {
	f, err := os.Open(key_filename)
	if err != nil {
		return
	}
	defer f.Close()

	buf, _ := ioutil.ReadAll(f)
	signer, _ = ssh.ParsePrivateKey(buf)
	return
}

func MakeKeyring(key_filenames []string) ssh.AuthMethod {
	signers := []ssh.Signer{}

	for _, key_filename := range key_filenames {
		signer, err := mkSigner(key_filename)
		if err == nil {
			signers = append(signers, signer)
		} else {
			log.Printf("Unable to create signer from %s (%s)", key_filename, err)
		}
	}

	return ssh.PublicKeys(signers...)
}

func (conn *SshConnection) Open(server string, auths ssh.AuthMethod, timeout int) error {
	s := strings.Split(server, "@")
	conn.server = s[1]
	// See if we have a port
	s2 := strings.Split(conn.server, ":")
	if len(s2) != 2 {
		conn.server += ":22"
	}
	conn.SudoCommand = "sudo -i /bin/bash -c"
	conn.config = &ssh.ClientConfig{
		User: s[0],
		Auth: []ssh.AuthMethod{auths},
	}
	conn.original_server = server
	conn.timeout = timeout
	return conn.open()
}

func (conn *SshConnection) open() error {
	conn.Close()
	network_conn, err := net.DialTimeout("tcp", conn.server, time.Duration(conn.timeout)*time.Second)
	if err != nil {
		return err
	}
	client_conn, new_chan, req_chan, err := ssh.NewClientConn(network_conn, conn.server, conn.config)
	if err != nil {
		network_conn.Close()
		return err
	}
	client := ssh.NewClient(client_conn, new_chan, req_chan)
	conn.network_conn = network_conn
	conn.client_conn = client_conn
	conn.client = client
	return err
}

func (conn *SshConnection) Close() {
	if conn.client_conn != nil {
		conn.client_conn.Close()
		conn.client_conn = nil
	}
	if conn.network_conn != nil {
		conn.network_conn.Close()
		conn.network_conn = nil
	}
}

func (conn *SshConnection) NewSession() (*ssh.Session, error) {
	sess, err := conn.client.NewSession()
	if err != nil { // Reopen and try again
		err = conn.open()
		if err == nil {
			sess, err = conn.client.NewSession()
		}
	}
	return sess, err
}

func (conn *SshConnection) Run(command string, timeout int, sudo bool) (*string, *string, error) {

	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")

	err := conn.RunWithWriters(command, timeout, sudo, stdout, stderr)
	stdout_s := stdout.String()
	stderr_s := stderr.String()
	return &stdout_s, &stderr_s, err
}

func (conn *SshConnection) RunWithWriters(command string, timeout int, sudo bool, stdout io.Writer, stderr io.Writer) error {
	session, err := conn.NewSession()
	if err != nil {
		log.Printf("Unable to open session: %s", err.Error())
		return err
	}
	defer session.Close()
	if timeout == 0 {
		conn.network_conn.SetDeadline(time.Time{})
	} else {
		conn.network_conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	}
	cmd := command
	if sudo {
		cmd = conn.SudoCommand + " " + Shellescape(command)
	}
	session.Stdout = stdout
	session.Stderr = stderr
	err = session.Run(cmd)
	return err
}
