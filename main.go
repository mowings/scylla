package main

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
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
}

const (
	NORMAL = iota
	SUDO   = iota
	SUDO_I = iota
)

const NO_TIMEOUT = 0

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

func mkKeyring(key_filenames []string) ssh.AuthMethod {
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

func (conn *SshConnection) Run(command string, timeout int, sudo int) (*string, *string, error) {
	session, err := conn.NewSession()
	if err != nil {
		log.Printf("Unable to open session: %s", err.Error())
		return nil, nil, err
	}
	defer session.Close()
	if timeout == 0 {
		conn.network_conn.SetDeadline(time.Time{})
	} else {
		conn.network_conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	}

	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")
	session.Stdout = stdout
	session.Stderr = stderr
	err = session.Run(command)
	stdout_s := stdout.String()
	stderr_s := stderr.String()
	return &stdout_s, &stderr_s, err
}

func main() {
	log.Printf("Starting ssh test...")
	auths := mkKeyring([]string{"keys/tron"})
	var conn SshConnection
	err := conn.Open("tron@devmo.hero3d.net:22", auths, 5)
	if err != nil {
		panic("Unable to connect: " + err.Error())
	}
	defer conn.Close()
	log.Println("Running command 1")
	stdout, stderr, err := conn.Run("sleep 5; ls -la", 1, NORMAL)
	if err != nil {
		log.Println("Failed to run: " + err.Error())
	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}
	log.Println("Running command 2")
	if stdout, stderr, err = conn.Run("df -h", 0, NORMAL); err != nil {
		log.Println("Failed to run: " + err.Error())
	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}
}
