package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/pkg/stdcopy"
	dockerContainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/pwnden/platform/internal/challenge"
)

// Official multi-architecture image. The connector has no host mounts, socket,
// published ports or external network. It only bridges a Docker exec byte stream
// to one consumer-selected service address, independent of the author's language.
const connectorImage = "busybox:1.37.0-musl@sha256:5cec3fc171c87218698e85a52af7087de727372aae264a787b8112901a5b0092"

var connectorMu sync.Mutex

func DialEndpoint(ctx context.Context, c *challenge.Loaded, name, instance string) (net.Conn, error) {
	state, err := ReadState(c)
	if err != nil {
		return nil, err
	}
	addresses, err := EndpointAddresses(ctx, c)
	if err != nil {
		return nil, err
	}
	valid := false
	for _, address := range addresses {
		if address.Name == name && address.Proxied && address.Instance == instance {
			valid = true
		}
	}
	if !valid {
		return nil, errors.New("HTTP endpoint is unavailable or belongs to an earlier run")
	}
	var endpoint challenge.Endpoint
	for _, candidate := range c.Endpoints {
		if candidate.Name == name {
			endpoint = candidate
		}
	}
	network, err := networkID(ctx, c, state.Project)
	if err != nil {
		return nil, err
	}
	engine, err := terminalEngine()
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			engine.Close()
		}
	}()
	out, stderr, err := command(ctx, c.Dir, []string{"FLAG=" + state.Flag}, "docker",
		composeArgs(c, state.Project, []string{c.Compose}, "ps", "-q", endpoint.Service)...)
	if err != nil {
		return nil, fmt.Errorf("find endpoint service: %w: %s", err, stderr)
	}
	ids := strings.Fields(out)
	if len(ids) != 1 {
		return nil, errors.New("endpoint requires one running service container")
	}
	inspected, err := engine.ContainerInspect(ctx, ids[0], client.ContainerInspectOptions{})
	if err != nil {
		return nil, err
	}
	service := inspected.Container
	if service.Config == nil || service.State == nil || !service.State.Running || service.NetworkSettings == nil ||
		service.Config.Labels["com.docker.compose.project"] != state.Project || service.Config.Labels["com.docker.compose.service"] != endpoint.Service {
		return nil, errors.New("endpoint service does not belong to this running problem")
	}
	address := ""
	for _, settings := range service.NetworkSettings.Networks {
		if settings.NetworkID == network {
			address = settings.IPAddress.String()
		}
	}
	if net.ParseIP(address) == nil {
		return nil, errors.New("endpoint service is not connected to the solve network")
	}
	id, err := ensureConnector(ctx, c, engine, state.Project, network)
	if err != nil {
		return nil, err
	}
	created, err := engine.ExecCreate(ctx, id, client.ExecCreateOptions{
		AttachStdin: true, AttachStdout: true, AttachStderr: true,
		Cmd: []string{"/bin/nc", address, strconv.Itoa(endpoint.Port)},
	})
	if err != nil {
		return nil, err
	}
	stream, err := engine.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return nil, err
	}
	local, remote := net.Pipe()
	// net.Pipe provides transport deadlines and unmodified binary bytes. Exec is
	// deliberately non-TTY; demultiplex stdout rather than applying terminal edits.
	go func() {
		_, _ = io.Copy(stream.Conn, remote)
		_ = stream.CloseWrite()
		stream.Close()
	}()
	go func() {
		_, _ = stdcopy.StdCopy(remote, io.Discard, stream.Reader)
		remote.Close()
		stream.Close()
		engine.Close()
	}()
	success = true
	return local, nil
}

func ensureConnector(ctx context.Context, c *challenge.Loaded, engine *client.Client, project, network string) (string, error) {
	connectorMu.Lock()
	defer connectorMu.Unlock()
	name := project + "-connector"
	existing, err := engine.ContainerInspect(ctx, name, client.ContainerInspectOptions{})
	if err == nil {
		if err := checkConnector(existing.Container, c, project, network); err != nil {
			return "", err
		}
		return existing.Container.ID, nil
	}
	if !errdefs.IsNotFound(err) {
		return "", err
	}
	if err := prepareToolImage(ctx, c, connectorImage); err != nil {
		return "", err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	claim := hex.EncodeToString(nonce)
	created, err := engine.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name: name,
		Config: &dockerContainer.Config{Image: connectorImage, User: "65534:65534", Entrypoint: []string{"/bin/sleep"}, Cmd: []string{"2147483647"},
			Labels: map[string]string{"com.docker.compose.project": project, "com.docker.compose.service": "pwnden-connector",
				"com.docker.compose.config-hash": "pwnden-connector-v1", "com.docker.compose.oneoff": "False",
				"pwnden.kind": "connector", "pwnden.repository": repositoryID(c.RepoRoot), "pwnden.claim": claim}},
		HostConfig: &dockerContainer.HostConfig{NetworkMode: dockerContainer.NetworkMode(network), ReadonlyRootfs: true,
			CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges"}},
	})
	if err == nil {
		_, err = engine.ContainerStart(ctx, created.ID, client.ContainerStartOptions{})
	}
	if err != nil {
		// A lost create response may still have created the container. The nonce
		// proves this call owns it before independent cleanup can remove it.
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		found, inspectErr := engine.ContainerInspect(cleanup, name, client.ContainerInspectOptions{})
		if inspectErr == nil && found.Container.Config != nil && found.Container.Config.Labels["pwnden.claim"] == claim {
			_, failure := engine.ContainerRemove(cleanup, found.Container.ID, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
			if failure != nil && !errdefs.IsNotFound(failure) {
				err = errors.Join(err, ErrCleanupFailed, failure)
			}
		} else if inspectErr != nil && !errdefs.IsNotFound(inspectErr) {
			err = errors.Join(err, ErrCleanupFailed, inspectErr)
		}
		return "", err
	}
	return created.ID, nil
}

func checkConnector(container dockerContainer.InspectResponse, c *challenge.Loaded, project, network string) error {
	cfg, host := container.Config, container.HostConfig
	if cfg == nil || host == nil || container.State == nil || !container.State.Running || container.NetworkSettings == nil ||
		cfg.Image != connectorImage || cfg.User != "65534:65534" || cfg.Labels["pwnden.kind"] != "connector" ||
		cfg.Labels["pwnden.repository"] != repositoryID(c.RepoRoot) || cfg.Labels["com.docker.compose.project"] != project ||
		strings.Join(cfg.Entrypoint, "\x00") != "/bin/sleep" || strings.Join(cfg.Cmd, "\x00") != "2147483647" ||
		!host.ReadonlyRootfs || host.Privileged || len(host.CapAdd) != 0 || strings.Join(host.CapDrop, "") != "ALL" ||
		strings.Join(host.SecurityOpt, "") != "no-new-privileges" || len(container.Mounts) != 0 || len(host.PortBindings) != 0 ||
		len(container.NetworkSettings.Networks) != 1 {
		return errors.New("existing connector does not match the isolated platform configuration")
	}
	for _, settings := range container.NetworkSettings.Networks {
		if settings.NetworkID != network {
			return errors.New("connector is connected to another network")
		}
	}
	return nil
}
