package e2e

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/random"

	"github.com/muchobien/lyra/internal/store"
)

func copyTree(test *testing.T, source, target string) {
	test.Helper()
	err := filepath.WalkDir(source, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(source, file)
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		test.Fatal(err)
	}
}

func TestBackupRestore(test *testing.T) {
	keys := test.TempDir()
	signingKey := newKey(test, "lyra-signing")
	secretsKey := filepath.Join(keys, "secrets")
	if err := os.WriteFile(secretsKey, []byte(strings.Repeat("5e", 32)+"\n"), 0o600); err != nil {
		test.Fatal(err)
	}
	flags := []string{"--signing-key-file", signingKey, "--secrets-key-file", secretsKey}
	first := filepath.Join(test.TempDir(), "first")
	srv := startServerAt(test, "127.0.0.1:"+freePort(test), first, "127.0.0.1", flags...)
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	push := func(dir string, args ...string) {
		test.Helper()
		mustGit(test, dir, append([]string{"-c", auth, "push", "-q", remote}, args...)...)
	}

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, ".forgejo/workflows/manual.yml", "on: workflow_dispatch\njobs:\n  echo:\n    runs-on: test\n    steps:\n      - env:\n          VALUE: ${{ secrets.KEY }}\n        run: echo \"length=${#VALUE}\"\n", "Add a manual workflow")
	push(src, "main")
	mustGit(test, src, "switch", "-q", "-c", "feature")
	commitFile(test, src, "feature.txt", "feature\n", "Add feature")
	push(src, "feature")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "feature", "--title", "Feature")
	if result := runLyra(test, srv, token, "pr", "merge", "acme/demo", "1", "--squash"); result.code != 0 {
		test.Fatalf("merge: %+v", result)
	}
	asset := filepath.Join(test.TempDir(), "asset.bin")
	if err := os.WriteFile(asset, bytes.Repeat([]byte("lyra"), 4096), 0o644); err != nil {
		test.Fatal(err)
	}
	releaseJSON(test, srv, token, "create", "acme/demo", "v1.0.0", "--asset", asset)
	lyraJSON(test, srv, token, &issueResult{}, "issue", "create", "acme/demo", "--title", "Kept across a restore")
	if status, body := call(test, http.MethodPut, srv.url+"/api/repos/acme/demo/secrets/KEY", bearer(token), `{"value":"hello"}`); status != http.StatusOK {
		test.Fatalf("secret: HTTP %d: %s", status, body)
	}
	image, _ := random.Image(1024, 1)
	if err := crane.Push(image, srv.address+"/acme/demo:v1", crane.Insecure, crane.WithAuth(&authn.Basic{Username: "lyra", Password: token})); err != nil {
		test.Fatal(err)
	}

	busy := filepath.Join(test.TempDir(), "busy")
	mustGit(test, "", "-c", auth, "clone", "-q", "-b", "main", remote, busy)
	mustGit(test, busy, "switch", "-q", "-c", "busy")
	var stop atomic.Bool
	pushed := make(chan int)
	go func() {
		count := 0
		for !stop.Load() {
			count++
			name := fmt.Sprintf("busy-%d.txt", count)
			os.WriteFile(filepath.Join(busy, name), []byte(name), 0o644)
			gitCommand(busy, nil, "add", name)
			gitCommand(busy, nil, "commit", "-q", "-m", name)
			if output, err := gitCommand(busy, nil, "-c", auth, "push", "-q", remote, "busy"); err != nil {
				test.Errorf("busy push: %v: %s", err, output)
				break
			}
		}
		pushed <- count
	}()
	eventually(test, "a few busy pushes", func() bool {
		output, _ := gitCommand(busy, nil, "rev-list", "--count", "busy", "^main")
		count, _ := strconv.Atoi(output)
		return count >= 3
	})

	file := filepath.Join(test.TempDir(), "backup.tar")
	result := runLyra(test, srv, token, "admin", "backup", "--output", file)
	stop.Store(true)
	pushes := <-pushed
	if result.code != 0 || !strings.Contains(result.stderr, `"pause_ms":`) {
		test.Fatalf("admin backup: %+v", result)
	}
	var status struct {
		LastBackup map[string]any `json:"last_backup"`
	}
	lyraJSON(test, srv, token, &status, "admin", "status")
	if status.LastBackup == nil || status.LastBackup["bytes"].(float64) < 1000 {
		test.Fatalf("status after a backup: %+v", status)
	}
	stat, _ := os.Stat(file)
	if float64(stat.Size()) != status.LastBackup["bytes"].(float64) {
		test.Fatalf("backup file has %d bytes, status says %v", stat.Size(), status.LastBackup["bytes"])
	}
	srv.stop()

	restored := filepath.Join(test.TempDir(), "restored")
	restore := func(args ...string) cliResult {
		test.Helper()
		command := exec.Command(lyraBinary, append([]string{"restore"}, args...)...)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		code := 0
		if exitError, ok := err.(*exec.ExitError); ok {
			code = exitError.ExitCode()
		}
		return cliResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
	}
	data, _ := os.ReadFile(file)
	truncated := filepath.Join(test.TempDir(), "truncated.tar")
	os.WriteFile(truncated, data[:len(data)/2], 0o644)
	if result := restore(truncated, "--data", filepath.Join(test.TempDir(), "x")); result.code != 1 {
		test.Fatalf("restore of a truncated backup: %+v", result)
	}
	tampered := filepath.Join(test.TempDir(), "tampered.tar")
	tamper(test, file, tampered)
	if result := restore(tampered, "--data", filepath.Join(test.TempDir(), "t")); result.code != 1 || !strings.Contains(result.stderr, "refs differ from the manifest") {
		test.Fatalf("restore with a manifest that doesn't match: %+v", result)
	}
	if result := restore(file, "--data", first); result.code != 1 || !strings.Contains(result.stderr, "isn't empty") {
		test.Fatalf("restore into a used directory: %+v", result)
	}
	var report struct {
		Repos   int   `json:"repos"`
		Objects int   `json:"objects"`
		Schema  int64 `json:"schema"`
	}
	result = restore(file, "--data", restored)
	if result.code != 0 || json.Unmarshal([]byte(result.stdout), &report) != nil || report.Repos != 1 || report.Objects != 2 || report.Schema < 13 {
		test.Fatalf("restore: %+v", result)
	}
	copyTree(test, filepath.Join(first, "blobs"), filepath.Join(restored, "blobs"))

	srv = startServerAt(test, "127.0.0.1:"+freePort(test), restored, "127.0.0.1", flags...)
	clone := filepath.Join(test.TempDir(), "clone")
	mustGit(test, "", "-c", auth, "clone", "-q", srv.url+"/acme/demo.git", clone)
	commits := strings.TrimSpace(mustGit(test, clone, "rev-list", "--count", "origin/busy", "^origin/main"))
	var events []store.Event
	status2, eventData := call(test, http.MethodGet, srv.url+"/api/events?repo=acme/demo&limit=1000", bearer(token), "")
	if status2 != http.StatusOK || json.Unmarshal(eventData, &events) != nil {
		test.Fatalf("events after restore: HTTP %d", status2)
	}
	busyPushes := 0
	for _, event := range events {
		if event.Action == "push" && strings.Contains(string(event.Data), "refs/heads/busy") {
			busyPushes++
		}
	}
	if strconv.Itoa(busyPushes) != commits || busyPushes == 0 || busyPushes > pushes {
		test.Fatalf("restored busy branch has %s commits and %d push events (%d pushes made)", commits, busyPushes, pushes)
	}

	var merged []struct {
		Signature struct {
			Status  string `json:"status"`
			KeyUser string `json:"key_user"`
		} `json:"signature"`
	}
	lyraJSON(test, srv, token, &merged, "commit", "list", "acme/demo", "--limit", "1")
	if merged[0].Signature.Status != "verified" || merged[0].Signature.KeyUser != "lyra" {
		test.Fatalf("squash merge after restore: %+v", merged)
	}
	if state := strings.TrimSpace(runLyra(test, srv, token, "pr", "view", "acme/demo", "1", "--jq", ".state").stdout); state != "merged" {
		test.Fatalf("pull request after restore: %s", state)
	}
	if status, body := call(test, http.MethodGet, srv.url+"/acme/demo/releases/latest/download/asset.bin", bearer(token), ""); status != http.StatusOK || len(body) != 4*4096 {
		test.Fatalf("asset after restore: HTTP %d, %d bytes", status, len(body))
	}
	if _, err := crane.Pull(srv.address+"/acme/demo:v1", crane.Insecure, crane.WithAuth(&authn.Basic{Username: "lyra", Password: token})); err != nil {
		test.Fatalf("image after restore: %v", err)
	}
	var issue struct {
		Number int64 `json:"number"`
	}
	lyraJSON(test, srv, token, &issue, "issue", "create", "acme/demo", "--title", "After the restore")
	if issue.Number != 3 {
		test.Fatalf("the next item number after restore is %d, want 3", issue.Number)
	}
	if result := runLyra(test, srv, token, "admin", "storage-check"); result.code != 0 || !strings.Contains(result.stdout, `"checked": 2`) {
		test.Fatalf("storage-check: %+v", result)
	}

	hostRunner(test, srv, token)
	if result := runLyra(test, srv, token, "workflow", "run", "acme/demo", "manual.yml"); result.code != 0 {
		test.Fatalf("dispatch after restore: %+v", result)
	}
	if result := runLyra(test, srv, token, "run", "wait", "acme/demo", "1", "--timeout", "60s"); result.code != 0 {
		test.Fatalf("run after restore: %+v", result)
	}
	if logs := runLyra(test, srv, token, "run", "logs", "acme/demo", "1").stdout; !strings.Contains(logs, "length=5") {
		test.Fatalf("the secret didn't decrypt after restore:\n%s", logs)
	}

	if err := os.RemoveAll(filepath.Join(restored, "blobs", "lyra", "releases")); err != nil {
		test.Fatal(err)
	}
	if result := runLyra(test, srv, token, "admin", "storage-check"); result.code != 1 || !strings.Contains(result.stdout, `"kind": "release_asset"`) {
		test.Fatalf("storage-check with a missing asset: %+v", result)
	}
	reader := createToken(test, srv, `{"name":"reader","scope":"read"}`)
	if result := runLyra(test, srv, reader, "admin", "backup", "--output", filepath.Join(test.TempDir(), "x.tar")); result.code != 1 || !strings.Contains(result.stderr, "unrestricted admin") {
		test.Fatalf("backup with a read token: %+v", result)
	}
}

func TestReadOnly(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	mustGit(test, src, "-c", auth, "push", "-q", remote, "main")
	limited := createToken(test, srv, `{"name":"limited","scope":"admin","repos":["acme/demo"]}`)
	if result := runLyra(test, srv, limited, "admin", "readonly", "on"); result.code != 1 {
		test.Fatalf("readonly with a repo token: %+v", result)
	}
	if result := runLyra(test, srv, token, "admin", "readonly", "on"); result.code != 0 || !strings.Contains(result.stdout, `"read_only": true`) {
		test.Fatalf("readonly on: %+v", result)
	}

	commitFile(test, src, "second.txt", "second\n", "Second commit")
	if output, err := gitCommand(src, nil, "-c", auth, "push", remote, "main"); err == nil || !strings.Contains(output, "503") {
		test.Fatalf("push while read-only: %v: %s", err, output)
	}
	if result := runLyra(test, srv, token, "issue", "create", "acme/demo", "--title", "x"); result.code != 1 || !strings.Contains(result.stderr, "read-only for maintenance") {
		test.Fatalf("API write while read-only: %+v", result)
	}
	if status, _, _ := postForm(test, srv.url+"/acme/demo/issues", url.Values{"title": {"x"}}, nil); status != http.StatusServiceUnavailable {
		test.Fatalf("UI form while read-only: HTTP %d", status)
	}
	image, _ := random.Image(256, 1)
	if err := crane.Push(image, srv.address+"/acme/demo:x", crane.Insecure, crane.WithAuth(&authn.Basic{Username: "lyra", Password: token})); err == nil || !strings.Contains(err.Error(), "UNAVAILABLE") && !strings.Contains(err.Error(), "503") {
		test.Fatalf("registry push while read-only: %v", err)
	}
	mustGit(test, "", "-c", auth, "clone", "-q", remote, filepath.Join(test.TempDir(), "clone"))
	if result := runLyra(test, srv, token, "repo", "get", "acme/demo"); result.code != 0 {
		test.Fatalf("read while read-only: %+v", result)
	}

	if result := runLyra(test, srv, token, "admin", "readonly", "off"); result.code != 0 || !strings.Contains(result.stdout, `"read_only": false`) {
		test.Fatalf("readonly off: %+v", result)
	}
	mustGit(test, src, "-c", auth, "push", "-q", remote, "main")
	status, data := call(test, http.MethodGet, srv.url+"/api/events?limit=50", bearer(token), "")
	if status != http.StatusOK || !strings.Contains(string(data), `"readonly.on"`) || !strings.Contains(string(data), `"readonly.off"`) {
		test.Fatalf("readonly events: HTTP %d", status)
	}
}

func tamper(test *testing.T, source, target string) {
	test.Helper()
	input, err := os.Open(source)
	if err != nil {
		test.Fatal(err)
	}
	defer input.Close()
	output, err := os.Create(target)
	if err != nil {
		test.Fatal(err)
	}
	defer output.Close()
	reader, writer := tar.NewReader(input), tar.NewWriter(output)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			test.Fatal(err)
		}
		content, _ := io.ReadAll(reader)
		if header.Name == "manifest.json" {
			var manifest map[string]any
			json.Unmarshal(content, &manifest)
			manifest["repos"].([]any)[0].(map[string]any)["refs"].(map[string]any)["refs/heads/main"] = strings.Repeat("0", 40)
			content, _ = json.Marshal(manifest)
			header.Size = int64(len(content))
		}
		writer.WriteHeader(header)
		writer.Write(content)
	}
	if err := writer.Close(); err != nil {
		test.Fatal(err)
	}
}

func TestMetrics(test *testing.T) {
	metricsAddress := "127.0.0.1:" + freePort(test)
	srv := startServerAt(test, "127.0.0.1:"+freePort(test), filepath.Join(test.TempDir(), "data"), "127.0.0.1", "--metrics-listen", metricsAddress)
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	createRepo(test, srv, token, "acme/other")
	scrape := func() string {
		test.Helper()
		var body []byte
		eventually(test, "the metrics listener", func() bool {
			response, err := http.Get("http://" + metricsAddress + "/metrics")
			if err != nil {
				return false
			}
			defer response.Body.Close()
			body, _ = io.ReadAll(response.Body)
			return response.StatusCode == http.StatusOK
		})
		return string(body)
	}
	first := scrape()
	for _, want := range []string{"lyra_repos 2\n", "lyra_read_only 0\n", `lyra_runs{status="success"} 0`, `lyra_jobs{status="waiting"} 0`, "lyra_runners_online 0\n", `lyra_http_requests_total{code="2xx",method="POST"}`, "go_goroutines", "process_resident_memory_bytes"} {
		if !strings.Contains(first, want) {
			test.Fatalf("metrics lack %q:\n%s", want, first)
		}
	}
	if strings.Contains(first, "lyra_backup_last_success_timestamp_seconds") {
		test.Fatal("a backup timestamp before any backup")
	}
	if result := runLyra(test, srv, token, "admin", "backup", "--output", filepath.Join(test.TempDir(), "backup.tar")); result.code != 0 {
		test.Fatalf("backup: %+v", result)
	}
	runLyra(test, srv, token, "admin", "readonly", "on")
	second := scrape()
	for _, want := range []string{"lyra_read_only 1\n", "lyra_backup_last_success_timestamp_seconds 1", "lyra_backup_last_bytes "} {
		if !strings.Contains(second, want) {
			test.Fatalf("metrics after a backup lack %q:\n%s", want, second)
		}
	}
	if status, body := call(test, http.MethodGet, srv.url+"/metrics", tailnet(), ""); status == http.StatusOK && strings.Contains(string(body), "lyra_repos") {
		test.Fatal("the main listener serves metrics")
	}
}
