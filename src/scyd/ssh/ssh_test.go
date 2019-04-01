package ssh

import (
	"log"
	"os"
	"testing"
)

var HOST = os.Getenv("TEST_HOST")
var KEYFILE = os.Getenv("TEST_KEYFILE")

func openConn() (conn SshConnection, err error) {
	// Use docker env values by default
	if HOST == "" {
		HOST = "scylla@localhost"
	}
	if KEYFILE == "" {
		KEYFILE = "/home/scylla/.ssh/scylla"
	}
	auths := MakeKeyring([]string{KEYFILE})
	err = conn.Open(HOST, auths, 5)
	if err != nil {
		panic("Unable to connect: " + err.Error())
	}
	return conn, err
}

func TestRegularAndSudo(t *testing.T) {
	conn, err := openConn()
	if err != nil {
		t.Error("Unable to open connection " + err.Error())
	}
	defer conn.Close()
	log.Println("Testing regular command...")
	if stdout, stderr, err := conn.Run("df -h", 0, false); err != nil {
		t.Error("Failed " + err.Error())

	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}
	log.Println("Testing sudo command and escaping...")
	if stdout, stderr, err := conn.Run("cat /var/log/syslog | grep 'pixelsquid' | wc -l", 0, true); err != nil {
		t.Error("Failed - " + err.Error())
	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}
}

func TestCommandTimeout(t *testing.T) {
	conn, err := openConn()
	if err != nil {
		t.Error("Unable to open connection " + err.Error())
	}
	defer conn.Close()
	if _, _, err := conn.Run("sleep 10", 2, false); err == nil {
		t.Error("Failed -- should have timed out after 2 seconds")
	}

	if stdout, stderr, err := conn.Run("uname -a", 2, false); err != nil {
		t.Error("Failed: " + err.Error())
	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}

}
