package dockere2e

import (
	"context"
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// gotta call this at the start or NONE of the flags work
	flag.Parse()

	// we need a client
	cli, err := GetClient()
	if err != nil {
		os.Exit(1)
	}

	// clean up the testing services that might have existed before we start
	CleanTestServices(context.Background(), cli)

	// run the tests, save the exit code
	exit := m.Run()
	// and then bow out
	os.Exit(exit)
}
