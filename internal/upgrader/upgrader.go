package upgrader

import "github.com/SymmetricalAI/symctl/internal/logger"

func Upgrade(dryRun bool) {
	logger.Debugf("Upgrade called with dry-run: %v\n", dryRun)
}
