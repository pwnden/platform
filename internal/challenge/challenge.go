package challenge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const SupportedContractVersion = 6

// Installed snapshots remain readable with the consumer's current isolation.
func supportedVersion(version int) bool { return version >= 1 && version <= SupportedContractVersion }

var ErrInvalidSlug = errors.New("invalid challenge slug")

// VersionError reports a contract the platform cannot consume.
type VersionError struct {
	Version    int
	Repository int
}

func (e *VersionError) Error() string {
	if e.Repository != 0 {
		return fmt.Sprintf("unsupported challenge contract version %d; repository uses %d and platform supports 1 through %d", e.Version, e.Repository, SupportedContractVersion)
	}
	return fmt.Sprintf("unsupported repository contract version %d; platform supports 1 through %d", e.Version, SupportedContractVersion)
}

// Contract contains the machine-readable values of the owner-published contract.
type Contract struct {
	Version             int    `toml:"version"`
	SolveNetwork        string `toml:"solve_network"`
	SolveTimeoutSeconds int    `toml:"solve_timeout_seconds"`
	AttackRejectedExit  int    `toml:"attack_rejected_exit"`
}

type Challenge struct {
	Schema     int        `toml:"schema"`
	Slug       string     `toml:"slug"`
	Title      any        `toml:"title"`
	Category   any        `toml:"category"`
	Difficulty int        `toml:"difficulty"`
	Files      []string   `toml:"files"`
	Compose    string     `toml:"compose"`
	Endpoints  []Endpoint `toml:"endpoints"`
	Flag       Flag       `toml:"flag"`
	Solve      Solve      `toml:"solve"`
	Patched    *Patched   `toml:"patched"`
	Content    Content    `toml:"content"`
	Player     Player     `toml:"player"`
}

type Player struct {
	Tools []string `toml:"tools"`
	CLI   []string `toml:"cli"`
}

func (c *Loaded) PlayerCLI() []string {
	if c.Schema < 6 {
		return []string{}
	}
	return append([]string{}, c.Player.CLI...)
}

// PlayerTools consumes author-selected tools; older catalogs derive a minimal set.
func (c *Loaded) PlayerTools() []string {
	if c.Schema >= 4 {
		return append([]string{}, c.Player.Tools...)
	}
	tools := []string{}
	web := false
	tcp := false
	for _, endpoint := range c.Endpoints {
		if endpoint.Protocol == "http" {
			web = true
		}
		if endpoint.Protocol == "tcp" {
			tcp = true
		}
	}
	if web {
		tools = append(tools, "web")
	}
	if len(c.Files) > 0 {
		tools = append(tools, "files")
	}
	if !web || tcp {
		tools = append(tools, "terminal")
	}
	return tools
}

type Content struct {
	Description string   `toml:"description"`
	Hints       []string `toml:"hints"`
	Walkthrough string   `toml:"walkthrough"`
}

type Endpoint struct {
	Name     string `toml:"name"`
	Service  string `toml:"service"`
	Port     int    `toml:"port"`
	Protocol string `toml:"protocol"`
}

type Flag struct {
	Mode   string `toml:"mode"`
	SHA256 string `toml:"sha256"`
}

type Solve struct {
	Image          string   `toml:"image"`
	Command        []string `toml:"command"`
	Network        string   `toml:"network"`
	TimeoutSeconds int      `toml:"timeout_seconds"`
	Writable       bool     `toml:"writable"`
}

type Patched struct {
	Compose string   `toml:"compose"`
	Check   []string `toml:"check"`
	Image   string   `toml:"image"`
}

type Loaded struct {
	Challenge
	Contract Contract
	RepoRoot string
	Dir      string
}

func Load(repo, slug string) (*Loaded, error) {
	return load(repo, slug, false)
}

// LoadForStop permits missing distribution files so cleanup remains available.
func LoadForStop(repo, slug string) (*Loaded, error) {
	return load(repo, slug, true)
}

func load(repo, slug string, cleanup bool) (*Loaded, error) {
	if !slugPattern.MatchString(slug) || len(slug) > 40 {
		return nil, fmt.Errorf("%w %q", ErrInvalidSlug, slug)
	}
	root, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository: %w", err)
	}
	definition, err := loadContract(root)
	if err != nil {
		return nil, err
	}
	dir, err := Within(root, filepath.Join(root, "challenges", slug))
	if err != nil {
		return nil, fmt.Errorf("resolve challenge: %w", err)
	}
	path, err := Resolve(root, dir, "challenge.toml")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var header struct {
		Schema int `toml:"schema"`
	}
	if err := toml.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("challenge.toml: %w", err)
	}
	if header.Schema != definition.Version {
		return nil, &VersionError{Version: header.Schema, Repository: definition.Version}
	}
	var c Challenge
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("challenge.toml: %w", err)
	}
	// Project identity comes from the already checked directory argument.
	c.Slug = slug
	if c.Solve.TimeoutSeconds == 0 {
		c.Solve.TimeoutSeconds = definition.SolveTimeoutSeconds
	}
	if c.Solve.Network == "" {
		c.Solve.Network = definition.SolveNetwork
	}
	if c.Compose != "" {
		if err := requireFile(root, dir, c.Compose); err != nil {
			return nil, fmt.Errorf("compose: %w", err)
		}
	}
	for _, file := range c.Files {
		if _, err := repositoryPath(root, dir, file); err != nil {
			return nil, fmt.Errorf("files %q: %w", file, err)
		}
		if _, err := Resolve(root, dir, file); err != nil {
			if cleanup && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("files %q: %w", file, err)
		}
	}
	if c.Patched != nil {
		if err := requireFile(root, dir, c.Patched.Compose); err != nil {
			return nil, fmt.Errorf("patched.compose: %w", err)
		}
	}
	return &Loaded{Challenge: c, Contract: definition, RepoRoot: root, Dir: dir}, nil
}

func loadContract(root string) (Contract, error) {
	var definition Contract
	path, err := Resolve(root, root, "contract.toml")
	if err != nil {
		return definition, fmt.Errorf("problem contract: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return definition, err
	}
	var header struct {
		Version int `toml:"version"`
	}
	if err := toml.Unmarshal(data, &header); err != nil {
		return definition, fmt.Errorf("contract.toml: %w", err)
	}
	if !supportedVersion(header.Version) {
		return definition, &VersionError{Version: header.Version}
	}
	if err := toml.Unmarshal(data, &definition); err != nil {
		return definition, fmt.Errorf("contract.toml: %w", err)
	}
	return definition, nil
}

func requireFile(root, base, name string) error {
	path, err := Resolve(root, base, name)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", name)
	}
	return nil
}

// Resolve accepts portable relative paths but keeps their real target in the repository.
func Resolve(root, base, name string) (string, error) {
	path, err := repositoryPath(root, base, name)
	if err != nil {
		return "", err
	}
	return Within(root, path)
}

func repositoryPath(root, base, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || filepath.VolumeName(name) != "" || strings.Contains(name, `\`) || strings.Contains(name, ":") {
		return "", fmt.Errorf("%q must be a relative repository path", name)
	}
	path := filepath.Join(base, filepath.FromSlash(name))
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%q is outside challenges repository", name)
	}
	return path, nil
}

func Within(root, path string) (string, error) {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%q is outside challenges repository", path)
	}
	return real, nil
}
