package ssh

import (
	"log"
	"testing"
)

func openConn() (conn SshConnection, err error) {
	auths := MakeKeyring([]string{"private_key"})
	err = conn.Open("tron@devmo.hero3d.net:22", auths, 5)
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
	if stdout, stderr, err := conn.Run("df -h", 0, NORMAL); err != nil {
		t.Error("Failed " + err.Error())

	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}
	log.Println("Testing sudo command and escaping...")
	if stdout, stderr, err := conn.Run("cat /var/log/syslog | grep 'pixelsquid' | wc -l", 0, SUDO_I); err != nil {
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
	if _, _, err := conn.Run("sleep 10", 2, NORMAL); err == nil {
		t.Error("Failed -- should have timed out after 2 seconds")
	}

	if stdout, stderr, err := conn.Run("uname -a", 2, NORMAL); err != nil {
		t.Error("Failed: " + err.Error())
	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}

}
