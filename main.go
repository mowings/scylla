package main

import (
	"github.com/mowings/scylla/ssh"
	"log"
)

func main() {
	log.Printf("Starting ssh test...")
	auths := ssh.MakeKeyring([]string{"keys/tron"})
	var conn ssh.SshConnection
	err := conn.Open("tron@devmo.hero3d.net:22", auths, 5)
	if err != nil {
		panic("Unable to connect: " + err.Error())
	}
	defer conn.Close()
	log.Println("Running command 1")
	stdout, stderr, err := conn.Run("cat /var/log/syslog | grep 'pixelsquid' | wc -l", 0, ssh.SUDO_I)
	if err != nil {
		log.Println("Failed to run: " + err.Error())
	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}
	log.Println("Running command 2")
	if stdout, stderr, err = conn.Run("df -h", 0, ssh.NORMAL); err != nil {
		log.Println("Failed to run: " + err.Error())
	} else {
		log.Println("\n" + *stdout + "\n===\n" + *stderr)
	}
}
