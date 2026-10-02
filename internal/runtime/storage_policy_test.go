package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pwnden/platform/internal/testutil"
)

func TestStoragePolicy(t *testing.T) {
	c := fixture(t)
	project := Project(c)
	raw := `{"services":{"app":{"volumes":[{"type":"bind","source":"` + c.Dir + `","target":"/input","read_only":true},{"type":"volume","source":"data","target":"/data"}],"tmpfs":["/run:size=99g"]}},"networks":{"default":{"name":"` + project + `_default"}},"volumes":{"data":{"name":"` + project + `_data"}}}`
	data, err := isolatedConfig(raw, c, project)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `size=256m,nosuid,nodev,nr_inodes=32768,mode=1777`) || !strings.Contains(string(data), `"nocopy":true`) || strings.Contains(string(data), "99g") {
		t.Fatal("storage quota not applied", string(data))
	}
	for _, changed := range []string{strings.Replace(raw, `"read_only":true`, `"read_only":false`, 1), strings.Replace(raw, `"tmpfs":`, `"scale":2,"tmpfs":`, 1)} {
		if _, err := isolatedConfig(changed, c, project); err == nil {
			t.Fatal("unsafe declaration accepted")
		}
	}
}

func TestToolboxImageVolumeRejectedBeforeCreation(t *testing.T) {
	c := fixture(t)
	c.Compose = ""
	calls := testutil.Docker(t,
		testutil.Reply{Match: []string{"image", "inspect"}, Out: "id"},
		testutil.Reply{Match: []string{"image", "inspect", "{{json .Config.Volumes}}"}, Out: `{"/data":{}}`},
	)
	_, err := RunTool(context.Background(), c, "", "image", []string{"true"})
	if err == nil || !strings.Contains(err.Error(), "anonymous VOLUME") {
		t.Fatal("anonymous image volume accepted", err)
	}
	for _, args := range calls() {
		if args[0] == "create" || args[0] == "start" {
			t.Fatal("unsafe image reached creation")
		}
	}
}

func TestExistingServiceDiskVolumeRejected(t *testing.T) {
	c := fixture(t)
	testutil.Docker(t, testutil.Reply{Match: []string{"volume", "inspect"}, Out: `[{"Driver":"local","Options":null}]`})
	err := checkServiceMounts(context.Background(), c, Project(c), map[string]Volume{"data": {Name: Project(c) + "_data"}})
	if err == nil || !strings.Contains(err.Error(), "no runtime quota") {
		t.Fatal("existing unbounded volume reused", err)
	}
}
