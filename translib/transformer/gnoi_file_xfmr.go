package transformer

import (
	"sync"
)

// Mutex for File RPCs.
var fileMu sync.Mutex

// FileRemove removes the specified file from the target.
func FileRemove(remoteFile string) (string, error) {
	fileMu.Lock()
	defer fileMu.Unlock()

	return checkQueryOutput(HostQuery("gpins_infra_host.exec_cmd", "rm "+remoteFile))
}
