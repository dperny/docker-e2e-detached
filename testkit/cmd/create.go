package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker-e2e/testkit/environment"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <cfg>",
	Short: "Provision a test environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("Config missing")
		}

		config, err := loadConfig(args[0])
		if err != nil {
			return err
		}

		var (
			env *environment.Environment
		)
		for r := 0; r < 100; r++ {
			t := time.Now()
			name := fmt.Sprintf("docker-e2e-%d%02d%02d-%d", t.Year(), t.Month(), t.Day(), r)
			env, err = environment.Provision(newSession(), name, config.Environment)
			if err != nil {
				// Try with another name.
				if strings.Contains(err.Error(), "AlreadyExistsException") {
					continue
				}
				return err
			}
			break
		}

		fmt.Println(env)
		return nil
	},
}
