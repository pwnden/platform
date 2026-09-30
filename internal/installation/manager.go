package installation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pwnden/platform/internal/challenge"
)

var ErrSetupRequired = errors.New("run pwnden setup first")

type Distribution struct {
	Archive  string
	Revision string
	SHA256   string
}

type Manager struct {
	DataDir      string
	Distribution Distribution
}

type receipt struct {
	Revision string `json:"revision"`
	SHA256   string `json:"sha256"`
}

func Default() (*Manager, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, err
	}
	return &Manager{DataDir: filepath.Join(base, "pwnden"), Distribution: Distribution{
		Archive: filepath.Join(filepath.Dir(executable), ArchiveName), Revision: catalogRevision, SHA256: catalogSHA256,
	}}, nil
}

func validHex(value string, size int) bool {
	_, err := hex.DecodeString(value)
	return len(value) == size && err == nil
}

func (m *Manager) directory(r receipt) (string, error) {
	if !validHex(r.Revision, 40) || !validHex(r.SHA256, 64) {
		return "", errors.New("invalid problem distribution identity")
	}
	return filepath.Join(m.DataDir, "catalogs", r.Revision+"-"+r.SHA256), nil
}

func readReceipt(path string) (receipt, error) {
	var r receipt
	data, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(data, &r)
	return r, err
}

func (m *Manager) Resolve() (string, error) {
	r, err := readReceipt(filepath.Join(m.DataDir, "installation.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrSetupRequired
		}
		return "", err
	}
	dir, err := m.directory(r)
	if err != nil {
		return "", err
	}
	installed, err := readReceipt(filepath.Join(dir, "receipt.json"))
	if err != nil || installed != r {
		return "", errors.Join(ErrSetupRequired, err)
	}
	root, err := m.root(dir)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(root, "contract.toml")); err != nil {
		return "", errors.Join(ErrSetupRequired, err)
	}
	return root, nil
}

func (m *Manager) root(directory string) (string, error) {
	base, err := filepath.EvalSymlinks(m.DataDir)
	if err != nil {
		return "", err
	}
	return challenge.Within(base, filepath.Join(directory, "catalog"))
}

type Pending struct {
	Root      string
	manager   *Manager
	receipt   receipt
	directory string
	created   bool
	active    bool
}

func (m *Manager) Stage(ctx context.Context) (_ *Pending, err error) {
	r := receipt{Revision: m.Distribution.Revision, SHA256: m.Distribution.SHA256}
	dir, err := m.directory(r)
	if err != nil {
		return nil, errors.New("this build has no valid problem distribution; run the checkout's ./pwnden setup or use a built platform package")
	}
	if installed, readErr := readReceipt(filepath.Join(dir, "receipt.json")); readErr == nil && installed == r {
		root, err := m.root(dir)
		if err != nil {
			return nil, err
		}
		return &Pending{Root: root, manager: m, receipt: r, directory: dir}, nil
	}
	archive, err := os.Open(m.Distribution.Archive)
	if err != nil {
		return nil, fmt.Errorf("open bundled problems; keep the built executable and catalog together: %w", err)
	}
	defer archive.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, &contextReader{ctx: ctx, reader: archive}); err != nil {
		return nil, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != r.SHA256 {
		return nil, errors.New("bundled problems checksum mismatch")
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	parent := filepath.Dir(dir)
	if info, err := os.Lstat(parent); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("managed catalog directory must not be a symlink")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(parent, 0700); err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(dir), ".install-")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(stage)) }()
	if err := extract(ctx, archive, stage); err != nil {
		return nil, err
	}
	if err := writeReceipt(filepath.Join(stage, "receipt.json"), r); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.Rename(stage, dir); err != nil {
		return nil, fmt.Errorf("install bundled problems: %w", err)
	}
	root, err := m.root(dir)
	if err != nil {
		return nil, errors.Join(err, os.RemoveAll(dir))
	}
	return &Pending{Root: root, manager: m, receipt: r, directory: dir, created: true}, nil
}

func (p *Pending) Activate() error {
	if err := writeReceipt(filepath.Join(p.manager.DataDir, "installation.json"), p.receipt); err != nil {
		return err
	}
	p.active = true
	return nil
}

func (p *Pending) Discard() error {
	if p.created && !p.active {
		return os.RemoveAll(p.directory)
	}
	return nil
}

func writeReceipt(path string, r receipt) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".receipt-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := json.NewEncoder(file).Encode(r); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
