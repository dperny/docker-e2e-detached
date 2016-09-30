package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"
)

const (
	region = "us-east-1"
)

type Config struct {
	Environment *EnvironmentConfig `yaml:"environment,omitempty"`

	Commands []string `yaml:"commands,omitempty"`
}

func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func runTests(c *Environment, cfg *Config) error {
	if err := c.Connect(); err != nil {
		return err
	}
	defer c.Disconnect()

	for _, cmd := range cfg.Commands {
		logrus.Infof("$ %s", cmd)
		now := time.Now()
		err := c.Run(cmd)
		duration := time.Since(now)
		if err != nil {
			logrus.Errorf("==> \"%s\" failed after %v: %s", cmd, duration, err)
			return err
		}
		logrus.Infof("==> \"%s\" completed in %v", cmd, duration)
	}

	return nil
}

var (
	cfg = &Config{
		Environment: &EnvironmentConfig{
			Template: "https://docker-for-aws.s3.amazonaws.com/aws/nightly/latest.json",

			SSHKeyName: "swarm",

			Managers: "3",
			Workers:  "5",

			InstanceType: "t2.micro",
		},

		Commands: []string{
			"docker version",
			"docker info",
			"docker pull dockerswarm/e2e",
			"docker run -v /var/run/docker.sock:/var/run/docker.sock --net=host dockerswarm/e2e",
		},
	}

	mainCmd = &cobra.Command{
		Use:   os.Args[0],
		Short: "Docker End to End Testing",
	}

	purgeCmd = &cobra.Command{
		Use:   "purge",
		Short: "Delete expired stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ttl, err := cmd.Flags().GetString("ttl")
			if err != nil {
				return err
			}
			ttlDelay, err := time.ParseDuration(ttl)
			if err != nil {
				return err
			}
			Purge(sess(), ttlDelay)
			return nil
		},
	}

	runCmd = &cobra.Command{
		Use:   "run <cfg>",
		Short: "Provision a test environment and run tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("Config missing")
			}

			config, err := loadConfig(args[0])
			if err != nil {
				return err
			}

			var (
				env *Environment
			)
			for r := 0; r < 100; r++ {
				t := time.Now()
				name := fmt.Sprintf("docker-e2e-%d%02d%02d-%d", t.Year(), t.Month(), t.Day(), r)
				env, err = Provision(sess(), name, config.Environment)
				if err != nil {
					// Try with another name.
					if strings.Contains(err.Error(), "AlreadyExistsException") {
						continue
					}
					return err
				}
				break
			}

			// Bring down the environment once we're done.
			defer env.Destroy()

			if err := runTests(env, config); err != nil {
				return err
			}

			return nil
		},
	}

	testCmd = &cobra.Command{
		Use:   "test <config> <environment>",
		Short: "Test an already provisioned environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("Config or AWS Stack ID missing")
			}

			config, err := loadConfig(args[0])
			if err != nil {
				return err
			}

			env := NewEnvironment(args[1], sess())

			if err := runTests(env, config); err != nil {
				return err
			}

			return nil
		},
	}
)

func sess() *session.Session {
	s, err := session.NewSession(aws.NewConfig().WithRegion(region))
	if err != nil {
		panic(err)
	}
	return s
}

func init() {
	purgeCmd.Flags().String("ttl", "1h", "Delete environments older than this")

	mainCmd.AddCommand(
		runCmd,
		testCmd,
		purgeCmd,
	)
}

func main() {
	if err := mainCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
