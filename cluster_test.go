package dockere2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

func TestClusterNodeAvailable(t *testing.T) {
	cli, err := GetClient()
	assert.NoError(t, err, "Client creation failed")

	// just list the nodes
	nodes, err := cli.NodeList(context.Background(), types.NodeListOptions{})
	assert.NoError(t, err, "Error listing nodes")

	leaders := 0
	for _, node := range nodes {
		// TODO(dperny) useful for debugging but not for production. remove
		// printNode(&node)
		if node.ManagerStatus != nil {
			// check that we have one leader among our nodes
			if node.ManagerStatus.Leader {
				leaders = leaders + 1
			}
			// check that all managers are reachable
			assert.Equal(t, swarm.ReachabilityReachable, node.ManagerStatus.Reachability, "node: %v", node.ID)
		}
		// check that all nodes are ready
		assert.Equal(t, swarm.NodeStateReady, node.Status.State, "node: %v", node.ID)
	}

	// covers both the case where there is no leader (more likely)
	// and if there are too many leaders (lol how even)
	assert.Equal(t, leaders, 1, "found wrong number of leaders")
}

func printNode(node *swarm.Node) {
	fmt.Printf("ID: %v\nManager: %v\nStatus:%v\n", node.ID, fmtMngr(node.ManagerStatus), node.Status)
}

func fmtMngr(m *swarm.ManagerStatus) string {
	if m.Leader {
		return "leader"
	} else {
		return string(m.Reachability)
	}
}
