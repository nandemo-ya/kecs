package phase2

import (
	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

// Suite-level shared resources
var (
	sharedKECS           utils.KECSContainerInterface
	sharedClient         utils.ECSClientInterface
	sharedLogger         *utils.TestLogger
	sharedClusterManager *utils.SharedClusterManager
)