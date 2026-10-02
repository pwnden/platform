package runtime

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pwnden/platform/internal/challenge"
	"github.com/tonistiigi/go-csvvalue"
)

type Config struct {
	Services map[string]Service  `json:"services"`
	Networks map[string]Network  `json:"networks"`
	Volumes  map[string]Volume   `json:"volumes"`
	Configs  map[string]Resource `json:"configs"`
	Secrets  map[string]Resource `json:"secrets"`
}

type Service struct {
	Image  string   `json:"image"`
	Tmpfs  []string `json:"tmpfs"`
	Scale  *int     `json:"scale"`
	Deploy struct {
		Replicas *int `json:"replicas"`
	} `json:"deploy"`
	Build        *Build          `json:"build"`
	Volumes      []Mount         `json:"volumes"`
	Networks     map[string]any  `json:"networks"`
	Privileged   bool            `json:"privileged"`
	UseAPISocket bool            `json:"use_api_socket"`
	Devices      []any           `json:"devices"`
	CapAdd       []string        `json:"cap_add"`
	PID          string          `json:"pid"`
	NetworkMode  string          `json:"network_mode"`
	Provider     json.RawMessage `json:"provider"`
	VolumesFrom  []string        `json:"volumes_from"`
	IPC          string          `json:"ipc"`
	Credential   json.RawMessage `json:"credential_spec"`
	Ports        []Port          `json:"ports"`
	GPUs         json.RawMessage `json:"gpus"`
	DeviceRules  []string        `json:"device_cgroup_rules"`
	PreStart     []Hook          `json:"pre_start"`
	PostStart    []Hook          `json:"post_start"`
	PreStop      []Hook          `json:"pre_stop"`
	SecurityOpt  []string        `json:"security_opt"`
	UsernsMode   string          `json:"userns_mode"`
	Cgroup       string          `json:"cgroup"`
	CgroupParent string          `json:"cgroup_parent"`
	Runtime      string          `json:"runtime"`
	UTS          string          `json:"uts"`
}

type Hook struct {
	Privileged bool `json:"privileged"`
}

type Port struct {
	Target   int    `json:"target"`
	Protocol string `json:"protocol"`
}

type Build struct {
	Context            string          `json:"context"`
	Dockerfile         string          `json:"dockerfile"`
	CacheFrom          []string        `json:"cache_from"`
	CacheTo            []string        `json:"cache_to"`
	AdditionalContexts json.RawMessage `json:"additional_contexts"`
	SSH                json.RawMessage `json:"ssh"`
	Privileged         bool            `json:"privileged"`
	Entitlements       []string        `json:"entitlements"`
}

type Mount struct {
	Target   string `json:"target"`
	Type     string `json:"type"`
	Source   string `json:"source"`
	ReadOnly bool   `json:"read_only"`
}

type Volume struct {
	Name       string         `json:"name"`
	External   bool           `json:"external"`
	Driver     string         `json:"driver"`
	DriverOpts map[string]any `json:"driver_opts"`
}

type Network struct {
	Name       string            `json:"name"`
	External   bool              `json:"external"`
	Driver     string            `json:"driver"`
	DriverOpts map[string]string `json:"driver_opts"`
}

type Resource struct {
	File     string `json:"file"`
	External bool   `json:"external"`
}

type State struct {
	Project string `json:"project"`
	Flag    string `json:"flag"`
}

var (
	ErrAlreadyRunning = errors.New("challenge is already running; stop it first")
	ErrNotRunning     = errors.New("challenge is not running")
	ErrCleanupFailed  = errors.New("resource cleanup failed")
)

func Project(c *challenge.Loaded) string {
	sum := sha256.Sum256([]byte(c.RepoRoot + "\x00" + c.Slug))
	return "pwnden-" + c.Slug + "-" + hex.EncodeToString(sum[:4])
}

func Validate(ctx context.Context, c *challenge.Loaded, flag string, patched bool) (*Config, error) {
	cfg, err := composeConfig(ctx, c, flag, patched)
	if err != nil || cfg == nil {
		return cfg, err
	}
	project := Project(c)
	if patched {
		project += "-patched"
	}
	if err := checkConfig(c, cfg, project); err != nil {
		return nil, err
	}
	return cfg, nil
}

func composeConfig(ctx context.Context, c *challenge.Loaded, flag string, patched bool) (*Config, error) {
	if c.Compose == "" {
		return nil, nil
	}
	out, err := resolvedConfig(ctx, c, flag, patched)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		return nil, fmt.Errorf("decode compose config: %w", err)
	}
	return &cfg, nil
}

func resolvedConfig(ctx context.Context, c *challenge.Loaded, flag string, patched bool) (string, error) {
	project := Project(c)
	files := []string{c.Compose}
	if patched {
		if c.Patched == nil {
			return "", errors.New("challenge has no patched compose")
		}
		project += "-patched"
		files = append(files, c.Patched.Compose)
	}
	out, stderr, err := command(ctx, c.Dir, []string{"FLAG=" + flag}, "docker", composeArgs(c, project, files, "config", "--format", "json")...)
	if err != nil {
		return "", fmt.Errorf("docker compose config: %w: %s", err, stderr)
	}
	return out, nil
}

func checkConfig(c *challenge.Loaded, cfg *Config, project string) error {
	if len(cfg.Services) == 0 {
		return errors.New("compose needs at least one service")
	}
	if _, ok := cfg.Networks[c.Solve.Network]; !ok {
		return fmt.Errorf("solve network %q is missing", c.Solve.Network)
	}
	if err := checkResources(cfg, project); err != nil {
		return err
	}
	if _, ok := cfg.Volumes["pwnden-workspace"]; ok {
		return errors.New("volume name pwnden-workspace is reserved for the temporary toolbox workspace")
	}
	if len(cfg.Volumes) > 8 {
		return errors.New("problem has too many temporary volumes")
	}
	for name, network := range cfg.Networks {
		for key, value := range network.DriverOpts {
			if (key != gatewayIPv4 && key != gatewayIPv6) || value != "isolated" {
				return fmt.Errorf("network %q requests unsupported driver option %q", name, key)
			}
		}
	}
	for name, service := range cfg.Services {
		if (service.Scale != nil && *service.Scale != 1) || (service.Deploy.Replicas != nil && *service.Deploy.Replicas != 1) {
			return fmt.Errorf("service %q must have one replica", name)
		}
		if len(service.Volumes)+len(service.Tmpfs) > 16 {
			return fmt.Errorf("service %q has too many mounts", name)
		}
		if service.Privileged || service.UseAPISocket || len(service.Devices) != 0 || len(service.CapAdd) != 0 ||
			service.PID != "" || (service.IPC != "" && service.IPC != "private" && service.IPC != "shareable") ||
			(service.NetworkMode != "" && service.NetworkMode != "none") || len(service.Provider) != 0 ||
			len(service.VolumesFrom) != 0 || len(service.Credential) != 0 || len(service.GPUs) != 0 || len(service.DeviceRules) != 0 ||
			service.UsernsMode != "" || (service.Cgroup != "" && service.Cgroup != "private") ||
			service.CgroupParent != "" || service.Runtime != "" || service.UTS != "" {
			return fmt.Errorf("service %q requests host privileges", name)
		}
		for _, hooks := range [][]Hook{service.PreStart, service.PostStart, service.PreStop} {
			for _, hook := range hooks {
				if hook.Privileged {
					return fmt.Errorf("service %q requests privileged hooks", name)
				}
			}
		}
		for _, option := range service.SecurityOpt {
			if !securityOptionAllowed(option) {
				return fmt.Errorf("service %q requests a custom security profile", name)
			}
		}
		for _, mount := range service.Volumes {
			switch mount.Type {
			case "bind":
				if !mount.ReadOnly {
					return fmt.Errorf("service %q requires read-only repository binds; use temporary volumes for writes", name)
				}
				if mount.Source == "" {
					return fmt.Errorf("service %q has bind mount without source", name)
				}
				if _, err := challenge.Within(c.RepoRoot, mount.Source); err != nil {
					return fmt.Errorf("service %q bind mount: %w", name, err)
				}
			case "volume", "tmpfs":
			default:
				return fmt.Errorf("service %q has unsupported mount type %q", name, mount.Type)
			}
		}
		if service.Build != nil {
			if service.Build.Context == "" {
				return fmt.Errorf("service %q has build without context", name)
			}
			contextPath, err := challenge.Within(c.RepoRoot, service.Build.Context)
			if err != nil {
				return fmt.Errorf("service %q build context: %w", name, err)
			}
			if service.Build.Dockerfile != "" {
				dockerfile := service.Build.Dockerfile
				if !filepath.IsAbs(dockerfile) {
					dockerfile = filepath.Join(contextPath, dockerfile)
				}
				if _, err := challenge.Within(c.RepoRoot, dockerfile); err != nil {
					return fmt.Errorf("service %q Dockerfile: %w", name, err)
				}
			}
			if len(service.Build.AdditionalContexts) != 0 {
				return fmt.Errorf("service %q additional build contexts are unsupported", name)
			}
			if err := checkCachePaths(c, service.Build.CacheFrom, "src"); err != nil {
				return fmt.Errorf("service %q cache_from: %w", name, err)
			}
			if err := checkCachePaths(c, service.Build.CacheTo, "dest"); err != nil {
				return fmt.Errorf("service %q cache_to: %w", name, err)
			}
			if len(service.Build.SSH) != 0 || service.Build.Privileged || len(service.Build.Entitlements) != 0 {
				return fmt.Errorf("service %q requests host build access", name)
			}
		}
	}
	for kind, resources := range map[string]map[string]Resource{"config": cfg.Configs, "secret": cfg.Secrets} {
		for name, resource := range resources {
			if resource.External {
				return fmt.Errorf("%s %q cannot be external", kind, name)
			}
			if resource.File != "" {
				if _, err := challenge.Within(c.RepoRoot, resource.File); err != nil {
					return fmt.Errorf("%s %q file: %w", kind, name, err)
				}
			}
		}
	}
	for _, endpoint := range c.Endpoints {
		if _, ok := cfg.Services[endpoint.Service]; !ok {
			return fmt.Errorf("endpoint %q refers to unknown service %q", endpoint.Name, endpoint.Service)
		}
	}
	return nil
}

func checkResources(cfg *Config, project string) error {
	for name, service := range cfg.Services {
		if len(service.Provider) != 0 {
			return fmt.Errorf("service %q requests a host provider", name)
		}
	}
	for name, network := range cfg.Networks {
		if network.External || network.Name != project+"_"+name || (network.Driver != "" && network.Driver != "bridge") {
			return fmt.Errorf("network %q must be a project-scoped bridge network", name)
		}
	}
	for name, volume := range cfg.Volumes {
		if volume.External || volume.Name != project+"_"+name || (volume.Driver != "" && volume.Driver != "local") || len(volume.DriverOpts) != 0 {
			return fmt.Errorf("volume %q must be a project-scoped local volume without driver_opts", name)
		}
	}
	return nil
}

func checkCachePaths(c *challenge.Loaded, entries []string, attribute string) error {
	for _, entry := range entries {
		// Cache references without attributes use the registry shorthand.
		if !strings.Contains(entry, "=") {
			continue
		}
		fields, err := csvvalue.Fields(entry, nil)
		if err != nil {
			return fmt.Errorf("invalid cache attributes: %w", err)
		}
		attributes := make(map[string]string, len(fields))
		for _, field := range fields {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				return fmt.Errorf("invalid cache attribute %q", field)
			}
			// Buildx treats keys case-insensitively and the last value wins.
			attributes[strings.ToLower(key)] = value
		}
		if attributes["type"] != "local" {
			continue
		}
		path := attributes[attribute]
		if path == "" {
			return fmt.Errorf("local cache requires %s", attribute)
		}
		if !filepath.IsAbs(path) {
			if filepath.VolumeName(path) != "" || os.IsPathSeparator(path[0]) {
				return fmt.Errorf("local cache %s %q must be absolute or relative to the problem directory", attribute, path)
			}
			// Keep parent components until symlinks have been resolved.
			path = c.Dir + string(os.PathSeparator) + path
		}
		// Builders use both the raw path and joined child paths. Check both
		// interpretations, resolving symlinks before each parent traversal.
		for _, candidate := range []string{path, filepath.Clean(path)} {
			if err := checkCachePath(c.RepoRoot, candidate); err != nil {
				return fmt.Errorf("local cache %s %q: %w", attribute, path, err)
			}
		}
	}
	return nil
}

func checkCachePath(root, path string) error {
	inside := func(target string) bool {
		relative, err := filepath.Rel(root, target)
		return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) && !filepath.IsAbs(relative)
	}
	volume := filepath.VolumeName(path)
	current := volume + string(os.PathSeparator)
	parts := strings.FieldsFunc(strings.TrimPrefix(path, volume), func(char rune) bool {
		return char == rune(os.PathSeparator) || (os.PathSeparator == '\\' && char == '/')
	})
	for _, part := range parts {
		switch part {
		case ".":
			continue
		case "..":
			current = filepath.Dir(current)
			continue
		}
		next := filepath.Join(current, part)
		resolved, err := filepath.EvalSymlinks(next)
		if err == nil {
			current = resolved
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		// An existing but dangling symlink cannot be treated as a new directory.
		if _, statErr := os.Lstat(next); statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
			return err
		}
		// Creating even an intermediate cache directory must stay in the repo.
		if !inside(next) {
			return fmt.Errorf("%q is outside challenges repository", next)
		}
		current = next
	}
	if !inside(current) {
		return fmt.Errorf("%q is outside challenges repository", current)
	}
	return nil
}

func Run(ctx context.Context, c *challenge.Loaded) error {
	if c.Compose == "" {
		return nil
	}
	if _, err := ReadState(c); err == nil {
		return ErrAlreadyRunning
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	flag, err := newFlag()
	if err != nil {
		return err
	}
	if _, err := Validate(ctx, c, flag, false); err != nil {
		return err
	}
	project := Project(c)
	err = startIsolated(ctx, c, flag, false)
	if err != nil {
		return errors.Join(err, Down(context.Background(), c, project, flag, false))
	}
	if err := SaveState(c, State{Project: project, Flag: flag}); err != nil {
		return errors.Join(err, Down(context.Background(), c, project, flag, false))
	}
	return nil
}

func Stop(ctx context.Context, c *challenge.Loaded) (err error) {
	defer func() {
		if err != nil && !errors.Is(err, ErrCleanupFailed) {
			err = errors.Join(ErrCleanupFailed, err)
		}
	}()
	if c.Compose == "" {
		return removeWorkspace(context.WithoutCancel(ctx), c, Project(c))
	}
	state, err := ReadState(c)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	flag := "pwnden{stopped}"
	if err == nil {
		flag = state.Flag
	}
	project := Project(c)
	// Cleanup completes even when the initiating command receives an interrupt.
	ctx = context.WithoutCancel(ctx)
	var cleanupErr error
	if c.Patched != nil {
		cleanupErr = Down(ctx, c, project+"-patched", flag, true)
	}
	cleanupErr = errors.Join(cleanupErr, Down(ctx, c, project, flag, false))
	if cleanupErr != nil {
		return cleanupErr
	}
	path, err := statePath(c)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func Down(ctx context.Context, c *challenge.Loaded, project, flag string, patched bool) (err error) {
	defer func() {
		if err != nil {
			err = errors.Join(ErrCleanupFailed, err)
		}
	}()
	expected := Project(c)
	if patched {
		expected += "-patched"
	}
	if project != expected {
		return fmt.Errorf("cleanup project %q does not match %q", project, expected)
	}
	downCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cfg, err := composeConfig(downCtx, c, flag, patched)
	if err != nil {
		return fmt.Errorf("validate cleanup for %s: %w", project, err)
	}
	if cfg == nil {
		return removeWorkspace(downCtx, c, project)
	}
	// Down removes project resources; deleted startup inputs must not block it.
	if err := checkResources(cfg, project); err != nil {
		return fmt.Errorf("validate cleanup for %s: %w", project, err)
	}
	files := []string{c.Compose}
	if patched {
		files = append(files, c.Patched.Compose)
	}
	_, stderr, err := command(downCtx, c.Dir, []string{"FLAG=" + flag}, "docker", composeArgs(c, project, files, "down", "--volumes", "--remove-orphans")...)
	if err != nil {
		return fmt.Errorf("docker compose down %s: %w: %s", project, err, stderr)
	}
	return removeWorkspace(downCtx, c, project)
}

func StartPatched(ctx context.Context, c *challenge.Loaded, flag string) (string, error) {
	if _, err := Validate(ctx, c, flag, true); err != nil {
		return "", err
	}
	project := Project(c) + "-patched"
	err := startIsolated(ctx, c, flag, true)
	if err != nil {
		return "", errors.Join(err, Down(context.Background(), c, project, flag, true))
	}
	return project, nil
}

// ToolExitError reports an ordinary nonzero exit from a completed solution.
// Docker launch failures, signals and cancellation are execution failures instead.
type ToolExitError struct {
	Code   int
	Stderr string
}

func (e *ToolExitError) Error() string { return fmt.Sprintf("toolbox exited %d: %s", e.Code, e.Stderr) }

func RunTool(ctx context.Context, c *challenge.Loaded, project, image string, args []string) (string, error) {
	out, _, err := runTool(ctx, c, project, image, args)
	return out, err
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunCommand distinguishes a completed user command from a runtime failure.
func RunCommand(ctx context.Context, c *challenge.Loaded, project, image string, args []string) (CommandResult, error) {
	out, stderr, err := runTool(ctx, c, project, image, args)
	if err == nil {
		return CommandResult{Stdout: out, Stderr: stderr}, nil
	}
	if exit, ok := err.(*ToolExitError); ok {
		return CommandResult{Stdout: out, Stderr: stderr, ExitCode: exit.Code}, nil
	}
	return CommandResult{}, err
}

func runTool(ctx context.Context, c *challenge.Loaded, project, image string, args []string) (string, string, error) {
	// A cold image download uses the caller's deadline, not the solution's runtime limit.
	if err := prepareToolImage(ctx, c, image); err != nil {
		return "", "", err
	}
	id := make([]byte, 8)
	if _, err := rand.Read(id); err != nil {
		return "", "", err
	}
	container := "pwnden-tool-" + hex.EncodeToString(id)
	network := "none"
	if c.Compose != "" {
		var err error
		network, err = networkID(ctx, c, project)
		if err != nil {
			return "", "", err
		}
	}
	mount := toolboxMount(c.Dir, c.Solve.Writable)
	if c.Solve.Writable {
		if project == "" {
			project = Project(c)
		}
		volume, err := ensureWorkspace(ctx, c, project)
		if err != nil {
			return "", "", err
		}
		mount = "type=volume,source=" + volume + ",target=/challenge,volume-nocopy"
	}
	options := []string{"create", "--rm", "--name", container, "--label", managedLabel + "=true", "--label", "pwnden.kind=tool", "--network", network, "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--mount", mount, "--workdir", "/challenge"}
	options = append(options, toolOptions(c.Solve.Writable)...)
	options = append(options, "--", image)
	options = append(options, args...)
	err := admitCreation(ctx, toolboxCost(), func(ctx context.Context) error {
		_, stderr, err := command(ctx, c.Dir, nil, "docker", options...)
		if err != nil {
			return fmt.Errorf("create toolbox: %w: %s", err, stderr)
		}
		return nil
	})
	if err != nil {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		return "", "", errors.Join(err, removeTool(cleanupCtx, c, container))
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.Solve.TimeoutSeconds)*time.Second)
	defer cancel()
	out, stderr, err := command(ctx, c.Dir, nil, "docker", "start", "--attach", container)
	if err != nil {
		var exit *exec.ExitError
		if ctx.Err() == nil && errors.As(err, &exit) && exit.ExitCode() > 0 && exit.ExitCode() < 125 {
			// --rm already removed a container whose command completed.
			return out, stderr, &ToolExitError{Code: exit.ExitCode(), Stderr: stderr}
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		return out, stderr, errors.Join(fmt.Errorf("toolbox: %w: %s", errors.Join(err, ctx.Err()), stderr), removeTool(cleanupCtx, c, container))
	}
	return out, stderr, nil
}

func removeTool(ctx context.Context, c *challenge.Loaded, container string) error {
	_, stderr, err := command(ctx, c.Dir, nil, "docker", "rm", "-f", container)
	if err == nil {
		return nil
	}
	// A launch failure or --rm may have left no container to remove.
	out, _, lookupErr := command(ctx, c.Dir, nil, "docker", "container", "ls", "--all", "--filter", "name=^/"+container+"$", "--format", "{{.ID}}")
	if lookupErr == nil && strings.TrimSpace(out) == "" {
		return nil
	}
	return fmt.Errorf("remove toolbox %s: %w: %s", container, errors.Join(ErrCleanupFailed, err, lookupErr), stderr)
}

func toolboxMount(source string, writable bool) string {
	fields := []string{"type=bind", "source=" + source, "target=/challenge"}
	if !writable {
		fields = append(fields, "readonly")
	}
	var output strings.Builder
	writer := csv.NewWriter(&output)
	_ = writer.Write(fields)
	writer.Flush()
	return strings.TrimSuffix(output.String(), "\n")
}

func networkID(ctx context.Context, c *challenge.Loaded, project string) (string, error) {
	filters := []string{"network", "ls", "--no-trunc", "--filter", "label=com.docker.compose.project=" + project, "--filter", "label=com.docker.compose.network=" + c.Solve.Network, "--format", "{{.ID}}"}
	out, stderr, err := command(ctx, c.Dir, nil, "docker", filters...)
	if err != nil {
		return "", fmt.Errorf("find solve network: %w: %s", err, stderr)
	}
	ids := strings.Fields(out)
	if len(ids) != 1 {
		return "", fmt.Errorf("expected one solve network %q for %s, found %d", c.Solve.Network, project, len(ids))
	}
	if err := checkLiveNetworks(ctx, c, project); err != nil {
		return "", err
	}
	return ids[0], nil
}

func composeArgs(c *challenge.Loaded, project string, files []string, tail ...string) []string {
	args := []string{"compose", "--project-directory", c.Dir, "-p", project}
	for _, file := range files {
		args = append(args, "-f", filepath.Join(c.Dir, filepath.FromSlash(file)))
	}
	return append(args, tail...)
}

func command(ctx context.Context, dir string, extraEnv []string, name string, args ...string) (string, string, error) {
	return commandInput(ctx, dir, extraEnv, nil, name, args...)
}

func newFlag() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "pwnden{" + hex.EncodeToString(b) + "}", nil
}

func statePath(c *challenge.Loaded) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(c.RepoRoot + "\x00" + c.Slug))
	return filepath.Join(base, "pwnden", "runs", hex.EncodeToString(sum[:])+".json"), nil
}

func ReadState(c *challenge.Loaded) (State, error) {
	var state State
	path, err := statePath(c)
	if err != nil {
		return state, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = errors.Join(ErrNotRunning, err)
		}
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Project != Project(c) || state.Flag == "" {
		return state, errors.New("invalid stored run state")
	}
	return state, nil
}

func SaveState(c *challenge.Loaded, state State) error {
	path, err := statePath(c)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "run-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if err := json.NewEncoder(file).Encode(state); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
