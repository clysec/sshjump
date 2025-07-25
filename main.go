package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/gliderlabs/ssh"
)

var debug = false

func debugPrint(args ...interface{}) {
	if debug {
		fmt.Println(args...)
	}
}

func main() {
	var val string
	var ok bool

	if val, ok = os.LookupEnv("ALLOWED_HOSTS"); !ok || val == "" {
		log.Fatal("ALLOWED_HOSTS environment variable is not set")
	} else {
		fmt.Println("Allowed hosts:", val)
	}

	if dbg, ok := os.LookupEnv("DEBUG"); ok && dbg == "true" {
		fmt.Println("Debug mode is enabled")
		debug = true
	} else {
		debug = false
	}

	var sshd_port string
	if sshd_port, ok = os.LookupEnv("SSHD_PORT"); !ok || sshd_port == "" {
		sshd_port = "2222" // Default port if not set
	}
	fmt.Println("Using SSHD_PORT:", sshd_port)

	allowedHosts := strings.Split(val, ";")
	allowedHostsMap := make(map[string]string)
	for _, host := range allowedHosts {
		from := strings.TrimSpace(host)
		to := strings.TrimSpace(host)
		if strings.Contains(from, "::") {
			parts := strings.Split(from, "::")
			if len(parts) == 2 {
				from = parts[0] // Use the host part only
				to = parts[1]
			}
		}
		debugPrint("Mapping from:", from, "to:", to)
		allowedHostsMap[from] = to
	}

	debugPrint("Allowed hosts map:", allowedHostsMap)

	ssh.Handle(func(s ssh.Session) {
		subs := s.RawCommand()
		debugPrint("Received command:", subs)

		if subs != "" && !strings.Contains(subs, ":") {
			subs = subs + ":22" // Default port if not specified
		}

		if subs == "" || allowedHostsMap == nil || allowedHostsMap[subs] == "" {
			io.WriteString(s, "Invalid or missing target host\n")
			return
		}

		debugPrint("Connecting to target host:", allowedHostsMap[subs])

		client, err := net.Dial("tcp", allowedHostsMap[subs])
		if err != nil {
			io.WriteString(s, "Failed to connect to target\n")
			return
		}
		defer client.Close()

		transport := sshTransport{
			SrvChannel: client,
			CliChannel: s,
			ErrC:       make(chan error, 1),
		}

		go transport.copyFromChannel()
		go transport.copyToChannel()

		err = <-transport.ErrC
		if err != nil {
			debugPrint("Error during copy:", err)
			io.WriteString(s, "Error during copy: "+err.Error()+"\n")
			return
		}
	})

	fmt.Println("Listening on port", sshd_port)
	log.Fatal(ssh.ListenAndServe(":"+sshd_port, nil))
}

type sshTransport struct {
	SrvChannel io.ReadWriter
	CliChannel io.ReadWriter
	ErrC       chan error
}

func (c sshTransport) copyFromChannel() {
	_, err := io.Copy(c.CliChannel, c.SrvChannel)
	c.ErrC <- err
}

func (c sshTransport) copyToChannel() {
	_, err := io.Copy(c.SrvChannel, c.CliChannel)
	c.ErrC <- err
}
