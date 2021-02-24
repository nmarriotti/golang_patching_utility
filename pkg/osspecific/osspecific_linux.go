// +build linux

package osspecific

import (
	"fmt"
	"io/fs"
	"syscall"
)

func GetLinuxFileInfo(fileInfo fs.FileInfo) [3]string {
	file_sys := fileInfo.Sys()
	uid := fmt.Sprint(file_sys.(*syscall.Stat_t).Uid)
	gid := fmt.Sprint(file_sys.(*syscall.Stat_t).Gid)
	perms := fmt.Sprintf("%04o", fileInfo.Mode().Perm())

	return [3]string{uid, gid, perms}
}
