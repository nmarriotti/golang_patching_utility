// +build windows

package osspecific

import (
	"io/fs"
)

func GetLinuxFileInfo(fileInfo fs.FileInfo) [3]string {
	return [3]string{"0", "0", "0"}
}
