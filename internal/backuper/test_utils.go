package backuper

import (
	"fmt"

	"github.com/stretchr/testify/mock"
)

const testTargetPath = "/tmp/sc-gitaly-backup"

var testCtx = mock.AnythingOfType("context.backgroundCtx")

func testBinaryFinder(name string) (string, error) {
	return fmt.Sprintf("/usr/local/bin/%s", name), nil
}
