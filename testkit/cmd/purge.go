package cmd

import (
	"time"

	"github.com/docker/docker-e2e/testkit/environment"
	"github.com/spf13/cobra"
)

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Delete expired stacks",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO(aluzzardi): TTL should perhaps be stored as a tag when creating stacks.
		ttl, err := cmd.Flags().GetString("ttl")
		if err != nil {
			return err
		}
		ttlDelay, err := time.ParseDuration(ttl)
		if err != nil {
			return err
		}
		return environment.Purge(newSession(), ttlDelay)
	},
}

func init() {
	purgeCmd.Flags().String("ttl", "1h", "Delete environments older than this")
}
