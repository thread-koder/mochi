package identity

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

const cgroupRoot = "/sys/fs/cgroup"

type pidCgroupKind int

const (
	pidCgroupGone pidCgroupKind = iota
	pidCgroupHost
	pidCgroupPod
)

type pidCgroupResult struct {
	podUID string
	kind   pidCgroupKind
	// cgroupDir is the container-level path under cgroupRoot (pidCgroupPod only).
	// Its inode matches bpf_get_current_cgroup_id().
	cgroupDir string
}

// podUIDPattern matches Kubernetes cgroup paths containing a pod UID.
// systemd paths may use underscores instead of hyphens in the UUID.
var podUIDPattern = regexp.MustCompile(
	`(?i)(?:pod)?([0-9a-f]{8}[_-][0-9a-f]{4}[_-][0-9a-f]{4}[_-][0-9a-f]{4}[_-][0-9a-f]{12})`,
)

func cgroupWatchRoots() []string {
	candidates := []string{
		filepath.Join(cgroupRoot, "kubepods.slice"),
		filepath.Join(cgroupRoot, "kubepods"),
	}
	roots := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			roots = append(roots, candidate)
		}
	}
	if len(roots) == 0 {
		return []string{cgroupRoot}
	}
	return roots
}

func hostCgroupDir(cgroupPath string) string {
	return filepath.Join(cgroupRoot, strings.TrimPrefix(cgroupPath, "/"))
}

// pidCgroupLookup classifies a host pid's cgroup: pod-shaped, host/non-pod, or gone.
func pidCgroupLookup(pid uint32) pidCgroupResult {
	path := filepath.Join("/proc", fmt.Sprintf("%d", pid), "cgroup")
	file, err := os.Open(path)
	if err != nil {
		return pidCgroupResult{kind: pidCgroupGone}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// cgroup v2: "0::/path"
		// cgroup v1: "id:controller:/path"
		parts := strings.SplitN(line, ":", 3)
		cgroupPath := line
		if len(parts) == 3 {
			cgroupPath = parts[2]
		}
		if podUID := extractPodUID(cgroupPath); podUID != "" {
			return pidCgroupResult{
				podUID:    podUID,
				kind:      pidCgroupPod,
				cgroupDir: hostCgroupDir(cgroupPath),
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return pidCgroupResult{kind: pidCgroupGone}
	}
	return pidCgroupResult{kind: pidCgroupHost}
}

func extractPodUID(cgroupPath string) string {
	match := podUIDPattern.FindStringSubmatch(cgroupPath)
	if len(match) < 2 {
		return ""
	}
	return strings.ReplaceAll(match[1], "_", "-")
}

func inodeOf(path string) (uint64, bool) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return 0, false
	}
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return stat.Ino, true
}

func walkPodCgroupInodes(onInode func(inode uint64, podUID string)) {
	for _, root := range cgroupWatchRoots() {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || !entry.IsDir() {
				return nil
			}
			podUID := extractPodUID(path)
			if podUID == "" {
				return nil
			}
			inode, ok := inodeOf(path)
			if !ok {
				return nil
			}
			onInode(inode, podUID)
			return nil
		})
	}
}
