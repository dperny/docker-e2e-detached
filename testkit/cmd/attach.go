package cmd

import (
	"errors"
	"fmt"
	"strings"

	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/docker/docker-e2e/testkit/environment"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <env>",
	Short: "attach a local socket to the docker socket on a remote host",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("Environment missing")
		}

		env := environment.New(args[0], newSession())
		ssh, err := env.SSHEndpoint()
		if err != nil {
			return err
		}
		openTunnel(ssh)
		return nil
	},
}

// openTunnel opens a tunnel over ssh from a socket in the local directory to
// the docker socket on the cluster.
func openTunnel(ssh string) {
	fmt.Printf("opening tunnel to %v\n", ssh)
	// set identity file
	// TODO(dperny) use real identity
	id := "/Users/drewerny/.ssh/swarm.pem"

	// split off the port number from the hostname
	// host, port := splitHostPort(ssh)
	host := ssh
	// TODO(dperny) handle error
	dir, _ := os.Getwd()
	// TODO(dperny) add flag to specify local socket attach point
	// set the socket file we're using
	socket := dir + "/docker.sock"
	// TODO(dperny) build this command in a better way
	cmd := exec.Command("ssh", "-vnNT", "-i"+id, "-L"+socket+":/var/run/docker.sock", "docker@"+host)
	// wire up the command outputs to our ouputs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// set up channel for stop signals
	signals := make(chan os.Signal, 1)
	// register for SIGINT and SIGTERM, this is what we will die on
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	// start the command
	cmd.Start()
	// tell the user how we're connecting
	fmt.Printf("remote docker server is listening locally at %v\n", socket)
	// TODO(dperny) print how to do configure docker daemon for this
	// wait for a signal
	sig := <-signals
	// pass it down to the ssh command
	cmd.Process.Signal(sig)
	// clean up the left-over socket
	os.Remove(socket)
}

// splitHostPort the SSH host name from the port.
// returns the hostname and the port. if no port is
// present, returns 22.
func splitHostPort(ssh string) (string, string) {
	sshInfo := strings.Split(ssh, ":")
	if len(sshInfo) < 2 {
		return sshInfo[0], "22"
	}
	return sshInfo[0], sshInfo[1]
}
