package runtime

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	dockerContainer "github.com/moby/moby/api/types/container"
	"github.com/pwnden/platform/internal/challenge"
)

const workspaceSize = 256 << 20
const workspaceInodes = 32768
const workspacePolicy = "tmpfs-player-files-v2"
const workspaceOptions = "size=256m,nosuid,nodev,nr_inodes=32768,uid=10001,gid=10001,mode=0700"

func workspaceCost() resourceCost { return resourceCost{25e7, 512 << 20, 32, 1} }

func workspaceLabels(c *challenge.Loaded, project string) []string {
	return []string{managedLabel + "=true", "pwnden.kind=workspace", "pwnden.project=" + project, "pwnden.repository=" + repositoryID(c.RepoRoot), "pwnden.workspace-policy=" + workspacePolicy}
}

func ensureWorkspace(ctx context.Context, c *challenge.Loaded, project string) (string, error) {
	return ensureWorkspaceFor(ctx, c, project, true)
}

func ensureWorkspaceFor(ctx context.Context, c *challenge.Loaded, project string, player bool) (string, error) {
	if project != Project(c) && project != Project(c)+"-patched" {
		return "", errors.New("invalid workspace project")
	}
	if !player {
		project += "-verification"
	}
	if err := prepareToolImage(ctx, c, connectorImage); err != nil {
		return "", err
	}
	name, volume := project+"-workspace", project+"_pwnden-workspace"
	initializing := false
	err := withAdmission(ctx, func(ctx context.Context, check func(resourceCost) error) error {
		exists, err := namedContainer(ctx, name)
		if err != nil {
			return err
		}
		if exists {
			if err := inspectWorkspace(ctx, c, project); err != nil {
				return err
			}
			_, stderr, err := command(ctx, "", nil, "docker", "exec", name, "/bin/test", "-f", "/state/ready")
			if err != nil {
				return fmt.Errorf("workspace initialization incomplete; stop and restart: %w: %s", err, stderr)
			}
			return nil
		}
		if err := check(workspaceCost()); err != nil {
			return err
		}
		out, _, err := command(ctx, "", nil, "docker", "volume", "ls", "--filter", "name=^"+volume+"$", "--format", "{{.Name}}")
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "" {
			return errors.New("workspace volume already exists without its keeper; stop the environment before retrying")
		}
		source := c.Dir
		if player {
			source, err = playerFiles(c)
			if err != nil {
				return err
			}
			defer os.RemoveAll(source)
		}
		archive, err := workspaceArchive(source)
		if err != nil {
			return err
		}
		defer os.Remove(archive.Name())
		defer archive.Close()
		labels := workspaceLabels(c, project)
		initializing = true
		args := []string{"volume", "create", "--driver", "local", "--opt", "type=tmpfs", "--opt", "device=tmpfs", "--opt", "o=" + workspaceOptions}
		for _, label := range labels {
			args = append(args, "--label", label)
		}
		if _, stderr, err := command(ctx, "", nil, "docker", append(args, volume)...); err != nil {
			return fmt.Errorf("create workspace volume: %w: %s", err, stderr)
		}
		if err := checkWorkspaceVolume(ctx, c, project); err != nil {
			return err
		}
		args = []string{"create", "--name", name, "--network", "none", "--user", "10001:10001", "--read-only", "--cgroupns", "private", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--cpus", "0.25", "--memory", "512m", "--memory-swap", "512m", "--pids-limit", "32", "--log-driver", "local", "--log-opt", "max-size=10m", "--log-opt", "max-file=3", "--tmpfs", "/state:rw,noexec,nosuid,nodev,size=1m,nr_inodes=32,uid=10001,gid=10001,mode=0700", "--mount", "type=volume,source=" + volume + ",target=/challenge,volume-nocopy", "--entrypoint", "/bin/sleep"}
		for _, label := range labels {
			args = append(args, "--label", label)
		}
		args = append(args, connectorImage, "2147483647")
		if _, stderr, err := command(ctx, "", nil, "docker", args...); err != nil {
			return fmt.Errorf("create workspace keeper: %w: %s", err, stderr)
		}
		if _, stderr, err := command(ctx, "", nil, "docker", "start", name); err != nil {
			return fmt.Errorf("start workspace keeper: %w: %s", err, stderr)
		}
		if _, stderr, err := commandInput(ctx, "", nil, archive, "docker", "exec", "-i", name, "/bin/tar", "-x", "-f", "-", "-C", "/challenge"); err != nil {
			return fmt.Errorf("seed bounded workspace: %w: %s", err, stderr)
		}
		_, stderr, err := command(ctx, "", nil, "docker", "exec", name, "/bin/touch", "/state/ready")
		if err != nil {
			return fmt.Errorf("mark workspace ready: %w: %s", err, stderr)
		}
		return nil
	})
	if err != nil && initializing {
		err = errors.Join(err, withRuntimeLock(context.Background(), false, func(ctx context.Context, _ func(resourceCost) error) error {
			return removeWorkspaceUnlocked(ctx, c, project)
		}))
	}
	return volume, err
}

func namedContainer(ctx context.Context, name string) (bool, error) {
	out, _, err := command(ctx, "", nil, "docker", "container", "ls", "--all", "--filter", "name=^/"+name+"$", "--format", "{{.ID}}")
	return strings.TrimSpace(out) != "", err
}

func inspectWorkspace(ctx context.Context, c *challenge.Loaded, project string) error {
	out, _, err := command(ctx, "", nil, "docker", "container", "inspect", project+"-workspace")
	if err != nil {
		return err
	}
	var containers []dockerContainer.InspectResponse
	if err := json.Unmarshal([]byte(out), &containers); err != nil {
		return err
	}
	if len(containers) != 1 {
		return errors.New("incomplete workspace inspection")
	}
	item := containers[0]
	cfg, host := item.Config, item.HostConfig
	if cfg == nil || host == nil || item.State == nil || !item.State.Running ||
		cfg.Image != connectorImage || cfg.User != "10001:10001" || cfg.Labels["pwnden.project"] != project || cfg.Labels["pwnden.repository"] != repositoryID(c.RepoRoot) || cfg.Labels["pwnden.workspace-policy"] != workspacePolicy || cfg.Labels["pwnden.kind"] != "workspace" ||
		!host.ReadonlyRootfs || host.NetworkMode != "none" || host.Privileged || len(host.CapAdd) != 0 || strings.Join(host.CapDrop, "") != "ALL" || strings.Join(host.SecurityOpt, "") != "no-new-privileges" || host.CgroupnsMode != "private" || host.NanoCPUs != 25e7 || host.Memory != 512<<20 || host.MemorySwap != host.Memory || host.PidsLimit == nil || *host.PidsLimit != 32 || len(item.Mounts) != 1 || host.Tmpfs["/state"] != "rw,noexec,nosuid,nodev,size=1m,nr_inodes=32,uid=10001,gid=10001,mode=0700" || len(host.PortBindings) != 0 || strings.Join(cfg.Entrypoint, "") != "/bin/sleep" || strings.Join(cfg.Cmd, "") != "2147483647" {
		return errors.New("existing workspace does not match the bounded configuration; stop and restart")
	}
	if host.LogConfig.Type != "local" || host.LogConfig.Config["max-size"] != "10m" || host.LogConfig.Config["max-file"] != "3" || len(host.Tmpfs) != 1 || len(host.Devices) != 0 || len(host.DeviceRequests) != 0 || host.PidMode != "" || host.UTSMode != "" || host.UsernsMode != "" || host.IpcMode != "private" {
		return errors.New("workspace isolation or log policy changed")
	}
	for _, mounted := range item.Mounts {
		if mounted.Destination == "/state" && string(mounted.Type) == "tmpfs" {
			continue
		}
		if mounted.Destination != "/challenge" || string(mounted.Type) != "volume" || mounted.Name != project+"_pwnden-workspace" || !mounted.RW {
			return errors.New("workspace mount changed")
		}
	}
	return checkWorkspaceVolume(ctx, c, project)
}

func checkWorkspaceVolume(ctx context.Context, c *challenge.Loaded, project string) error {
	out, _, err := command(ctx, "", nil, "docker", "volume", "inspect", project+"_pwnden-workspace")
	if err != nil {
		return err
	}
	var volumes []struct {
		Driver          string
		Labels, Options map[string]string
	}
	if err := json.Unmarshal([]byte(out), &volumes); err != nil {
		return err
	}
	if len(volumes) != 1 || volumes[0].Driver != "local" || volumes[0].Labels["pwnden.project"] != project || volumes[0].Labels["pwnden.repository"] != repositoryID(c.RepoRoot) || volumes[0].Labels["pwnden.kind"] != "workspace" || volumes[0].Labels["pwnden.workspace-policy"] != workspacePolicy || volumes[0].Options["type"] != "tmpfs" || volumes[0].Options["device"] != "tmpfs" || volumes[0].Options["o"] != workspaceOptions || len(volumes[0].Options) != 3 {
		return errors.New("workspace volume ownership or quota changed")
	}
	return nil
}

func removeWorkspace(ctx context.Context, c *challenge.Loaded, project string) error {
	if !c.Solve.Writable {
		return nil
	}
	return withRuntimeLock(ctx, false, func(ctx context.Context, _ func(resourceCost) error) error {
		return errors.Join(removeWorkspaceUnlocked(ctx, c, project), removeWorkspaceUnlocked(ctx, c, project+"-verification"))
	})
}

func removeWorkspaceUnlocked(ctx context.Context, c *challenge.Loaded, project string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	name := project + "-workspace"
	out, _, err := command(ctx, "", nil, "docker", "container", "ls", "--all", "--filter", "name=^/"+name+"$", "--filter", "label=pwnden.kind=workspace", "--filter", "label=pwnden.project="+project, "--filter", "label=pwnden.repository="+repositoryID(c.RepoRoot), "--format", "{{.ID}}")
	if err != nil {
		return err
	}
	if ids := strings.Fields(out); len(ids) > 0 {
		if _, stderr, err := command(ctx, "", nil, "docker", append([]string{"rm", "-f", "-v"}, ids...)...); err != nil {
			return fmt.Errorf("remove workspace: %w: %s", err, stderr)
		}
	}
	out, _, err = command(ctx, "", nil, "docker", "volume", "ls", "--filter", "name=^"+project+"_pwnden-workspace$", "--filter", "label=pwnden.project="+project, "--filter", "label=pwnden.repository="+repositoryID(c.RepoRoot), "--filter", "label=pwnden.kind=workspace", "--format", "{{.Name}}")
	if err != nil {
		return err
	}
	if names := strings.Fields(out); len(names) > 0 {
		_, stderr, err := command(ctx, "", nil, "docker", append([]string{"volume", "rm"}, names...)...)
		if err != nil {
			return fmt.Errorf("remove workspace volume: %w: %s", err, stderr)
		}
	}
	return nil
}

func workspaceArchive(source string) (*os.File, error) {
	file, err := os.CreateTemp("", "pwnden-workspace-*.tar")
	if err != nil {
		return nil, err
	}
	writer := tar.NewWriter(file)
	var size int64
	entries := 0
	err = filepath.WalkDir(source, func(path string, entry os.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() && !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("workspace input is not a file, directory or symlink: %s", path)
		}
		size += info.Size()
		entries++
		if size > workspaceSize || entries > workspaceInodes {
			return errors.New("problem source exceeds the temporary workspace quota")
		}
		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			link, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		header.Name, header.Uid, header.Gid, header.Uname, header.Gname = filepath.ToSlash(relative), 10001, 10001, "", ""
		header.Mode = header.Mode&0777 | 0600
		if info.IsDir() {
			header.Mode |= 0700
		}
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			input, err := os.Open(path)
			if err != nil {
				return err
			}
			_, err = io.CopyN(writer, input, info.Size())
			input.Close()
			return err
		}
		return nil
	})
	err = errors.Join(err, writer.Close())
	if err == nil {
		_, err = file.Seek(0, 0)
	}
	if err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

// Validate actual mounts before processes start; image VOLUME declarations can
// otherwise allocate anonymous disk volumes outside the resolved Compose model.
func checkServiceMounts(ctx context.Context, c *challenge.Loaded, project string, volumes map[string]Volume) error {
	for _, volume := range volumes {
		out, _, err := command(ctx, c.Dir, nil, "docker", "volume", "inspect", volume.Name)
		if err != nil {
			return err
		}
		var actual []struct {
			Driver  string
			Options map[string]string
		}
		if err := json.Unmarshal([]byte(out), &actual); err != nil {
			return err
		}
		if len(actual) != 1 || actual[0].Driver != "local" || len(actual[0].Options) != 3 || actual[0].Options["type"] != "tmpfs" || actual[0].Options["device"] != "tmpfs" || actual[0].Options["o"] != "size=256m,nosuid,nodev,nr_inodes=32768,mode=1777" {
			return errors.New("existing service volume has no runtime quota; stop and restart")
		}
	}
	out, _, err := command(ctx, c.Dir, nil, "docker", "container", "ls", "--all", "--filter", "label=com.docker.compose.project="+project, "--format", "{{.ID}}")
	if err != nil {
		return err
	}
	ids := strings.Fields(out)
	if len(ids) == 0 {
		return errors.New("problem has no created services")
	}
	out, _, err = command(ctx, c.Dir, nil, "docker", append([]string{"container", "inspect"}, ids...)...)
	if err != nil {
		return err
	}
	var containers []dockerContainer.InspectResponse
	if err := json.Unmarshal([]byte(out), &containers); err != nil {
		return err
	}
	if len(containers) != len(ids) {
		return errors.New("incomplete service mount inspection")
	}
	for _, item := range containers {
		for _, mounted := range item.Mounts {
			switch string(mounted.Type) {
			case "bind":
				if mounted.RW {
					return errors.New("writable host bind is prohibited")
				}
			case "volume":
				allowed := false
				for _, volume := range volumes {
					if mounted.Name == volume.Name {
						allowed = true
					}
				}
				if !allowed {
					return errors.New("anonymous or undeclared service volumes are prohibited")
				}
			case "tmpfs":
			default:
				return errors.New("unsupported service mount")
			}
		}
	}
	return nil
}
