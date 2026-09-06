package identity

import (
	"context"
	"io/fs"
	"path/filepath"
	"unsafe"

	"github.com/thread_koder/mochi/agent/internal/logger"
	"golang.org/x/sys/unix"
)

const inotifyMask = uint32(unix.IN_CREATE | unix.IN_MOVED_TO | unix.IN_DELETE | unix.IN_ONLYDIR)

type watchDir struct {
	path  string
	inode uint64
}

// watchCgroups recursively inotify-watches kubelet cgroup dirs so container
// inodes are indexed as they appear and dropped when kubelet removes them.
func (r *Resolver) watchCgroups(ctx context.Context) {
	log := logger.WithComponent("identity")

	inotifyFD, err := unix.InotifyInit1(unix.IN_CLOEXEC)
	if err != nil {
		log.Warn().Err(err).Msg("cgroup inotify init failed, falling back to periodic rebuild")
		r.rebuild()
		return
	}
	defer unix.Close(inotifyFD)

	watchesByID := make(map[int]watchDir)
	addWatch := func(dir string) {
		inode, ok := inodeOf(dir)
		if !ok {
			return
		}
		watchID, err := unix.InotifyAddWatch(inotifyFD, dir, inotifyMask)
		if err != nil {
			return
		}
		watchesByID[watchID] = watchDir{path: dir, inode: inode}
	}

	for _, root := range cgroupWatchRoots() {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || !entry.IsDir() {
				return nil
			}
			addWatch(path)
			r.indexPath(path)
			return nil
		})
	}

	buf := make([]byte, 64*1024)
	for {
		if ctx.Err() != nil {
			return
		}
		pollFDs := []unix.PollFd{{Fd: int32(inotifyFD), Events: unix.POLLIN}}
		nready, err := unix.Poll(pollFDs, 500)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			log.Warn().Err(err).Msg("cgroup inotify poll failed")
			return
		}
		if nready == 0 {
			continue
		}

		nbytes, err := unix.Read(inotifyFD, buf)
		if err != nil {
			if err == unix.EINTR || err == unix.EAGAIN {
				continue
			}
			log.Warn().Err(err).Msg("cgroup inotify read failed")
			return
		}

		for offset := 0; offset+unix.SizeofInotifyEvent <= nbytes; {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			nameLen := int(event.Len)
			name := ""
			if nameLen > 0 {
				nameStart := offset + unix.SizeofInotifyEvent
				nameEnd := nameStart + nameLen
				if nameEnd > nbytes {
					break
				}
				name = unix.ByteSliceToString(buf[nameStart:nameEnd])
			}
			offset += unix.SizeofInotifyEvent + nameLen

			if event.Mask&unix.IN_Q_OVERFLOW != 0 {
				r.rebuild()
				continue
			}
			if event.Mask&unix.IN_IGNORED != 0 {
				if watched, ok := watchesByID[int(event.Wd)]; ok {
					r.unindex(watched.inode)
					delete(watchesByID, int(event.Wd))
				}
				continue
			}
			if name == "" {
				continue
			}
			parent, ok := watchesByID[int(event.Wd)]
			if !ok {
				continue
			}
			path := filepath.Join(parent.path, name)
			if event.Mask&unix.IN_ISDIR == 0 {
				continue
			}
			if event.Mask&unix.IN_DELETE != 0 {
				r.dropWatchPath(watchesByID, path)
				continue
			}
			addWatch(path)
			r.indexPath(path)
		}
	}
}

func (r *Resolver) indexPath(path string) {
	podUID := extractPodUID(path)
	if podUID == "" {
		return
	}
	r.mapsMu.RLock()
	_, known := r.podsByUID[podUID]
	r.mapsMu.RUnlock()
	if !known {
		return
	}
	inode, ok := inodeOf(path)
	if !ok {
		return
	}
	r.index(inode, podUID)
}

func (r *Resolver) dropWatchPath(watchesByID map[int]watchDir, path string) {
	for watchID, watched := range watchesByID {
		if watched.path != path {
			continue
		}
		r.unindex(watched.inode)
		delete(watchesByID, watchID)
		return
	}
}
