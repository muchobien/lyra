package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/muchobien/lyra/internal/store"
)

const runnerModule = "code.forgejo.org/forgejo/runner/v13"

const runnerVersion = "v13.2.1"

var (
	runnerOnce   sync.Once
	runnerPath   string
	runnerOutput []byte
)

func runnerBinary(test *testing.T) string {
	test.Helper()
	runnerOnce.Do(func() {
		dir := filepath.Dir(lyraBinary)
		install := exec.Command("go", "install", "-ldflags", "-X "+runnerModule+"/internal/pkg/ver.version="+runnerVersion, runnerModule+"@"+runnerVersion)
		install.Env = append(os.Environ(), "GOBIN="+dir)
		if runnerOutput, _ = install.CombinedOutput(); install.ProcessState.ExitCode() == 0 {
			runnerPath = filepath.Join(dir, "runner")
		}
	})
	if runnerPath == "" {
		test.Fatalf("install forgejo-runner %s: %s", runnerVersion, runnerOutput)
	}
	return runnerPath
}

type runnerProcess struct {
	config string
	logs   *lockedBuffer
}

func newRunner(test *testing.T, name string) runnerProcess {
	test.Helper()
	dir := filepath.Join(test.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		test.Fatal(err)
	}
	config := filepath.Join(dir, "config.yaml")
	content := "log:\n  level: debug\nrunner:\n  file: " + filepath.Join(dir, "registration.json") + "\n  fetch_interval: 1s\n  fetch_timeout: 5s\n"
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
	return runnerProcess{config: config, logs: &lockedBuffer{}}
}

func (runner runnerProcess) register(test *testing.T, srv server, token, name string, extra ...string) (string, error) {
	test.Helper()
	args := append([]string{"--config", runner.config, "register", "--no-interactive", "--instance", srv.url, "--token", token, "--name", name, "--labels", "test:host"}, extra...)
	command := exec.Command(runnerBinary(test), args...)
	command.Dir = filepath.Dir(runner.config)
	done := make(chan struct{})
	var output []byte
	var err error
	go func() {
		output, err = command.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
		return string(output), err
	case <-time.After(30 * time.Second):
		command.Process.Kill()
		<-done
		test.Fatalf("forgejo-runner register hung: %s", output)
		return "", nil
	}
}

func (runner runnerProcess) start(test *testing.T) {
	test.Helper()
	command := exec.Command(runnerBinary(test), "--config", runner.config, "daemon")
	command.Dir = filepath.Dir(runner.config)
	command.Stdout = runner.logs
	command.Stderr = runner.logs
	if err := command.Start(); err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		command.Process.Kill()
		command.Wait()
		if test.Failed() {
			test.Logf("runner %s log:\n%s", runner.config, runner.logs.String())
		}
	})
}

func eventually(test *testing.T, what string, check func() bool) {
	test.Helper()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		if check() {
			return
		}
	}
	test.Fatalf("timed out waiting for %s", what)
}

type runnerResult struct {
	store.Runner
	Online bool `json:"online"`
}

func TestRunners(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	limited := createToken(test, srv, `{"name":"limited","scope":"admin","repos":["acme/demo"]}`)
	runners := func() map[string]runnerResult {
		test.Helper()
		var list []runnerResult
		lyraJSON(test, srv, token, &list, "runner", "list")
		found := map[string]runnerResult{}
		for _, runner := range list {
			found[runner.Name] = runner
		}
		return found
	}

	var registration struct {
		ID     int64  `json:"id"`
		Secret string `json:"secret"`
		URL    string `json:"url"`
	}
	lyraJSON(test, srv, token, &registration, "runner", "token", "create", "--name", "e2e")
	if len(registration.Secret) != 40 || registration.URL != srv.url {
		test.Fatalf("registration token: %+v", registration)
	}
	if result := runLyra(test, srv, limited, "runner", "token", "create", "--name", "x"); result.code != 1 || !strings.Contains(result.stderr, "unrestricted admin") {
		test.Fatalf("token for one repo creates a registration token: %+v", result)
	}

	first := newRunner(test, "first")
	if output, err := first.register(test, srv, registration.Secret, "e2e-first"); err != nil {
		test.Fatalf("register: %v: %s", err, output)
	}
	first.start(test)
	eventually(test, "the runner to declare its labels", func() bool {
		runner := runners()["e2e-first"]
		return runner.Online && runner.Version == runnerVersion && strings.Join(runner.Labels, ",") == "test" && !runner.Ephemeral
	})
	seen := *runners()["e2e-first"].LastSeenAt
	eventually(test, "the runner to poll again", func() bool {
		return runners()["e2e-first"].LastSeenAt.After(seen)
	})

	second := newRunner(test, "second")
	if output, err := second.register(test, srv, registration.Secret, "e2e-ephemeral", "--ephemeral"); err != nil {
		test.Fatalf("register --ephemeral: %v: %s", err, output)
	}
	if runner := runners()["e2e-ephemeral"]; !runner.Ephemeral {
		test.Fatalf("ephemeral runner: %+v", runner)
	}
	bad := newRunner(test, "bad")
	if output, err := bad.register(test, srv, strings.Repeat("0", 40), "e2e-bad"); err == nil || !strings.Contains(output, "unknown registration token") {
		test.Fatalf("register with an unknown token: %v: %s", err, output)
	}

	if result := runLyra(test, srv, token, "runner", "token", "delete", fmt.Sprint(registration.ID)); result.code != 0 {
		test.Fatalf("delete registration token: %+v", result)
	}
	if output, err := bad.register(test, srv, registration.Secret, "e2e-late"); err == nil || !strings.Contains(output, "unknown registration token") {
		test.Fatalf("register with a deleted token: %v: %s", err, output)
	}
	if _, found := runners()["e2e-first"]; !found {
		test.Fatal("deleting the registration token removed its runners")
	}

	if result := runLyra(test, srv, token, "runner", "delete", fmt.Sprint(runners()["e2e-first"].ID)); result.code != 0 {
		test.Fatalf("delete runner: %+v", result)
	}
	eventually(test, "the deleted runner to be refused", func() bool {
		return strings.Contains(first.logs.String(), "unregistered runner")
	})
	var created struct {
		UUID  string `json:"uuid"`
		Token string `json:"token"`
		URL   string `json:"url"`
	}
	lyraJSON(test, srv, token, &created, "runner", "create", "--name", "e2e-configured")
	configured := newRunner(test, "configured")
	connection := "log:\n  level: debug\nrunner:\n  fetch_interval: 1s\nserver:\n  connections:\n    lyra:\n      url: " + created.URL + "\n      uuid: " + created.UUID + "\n      token: " + created.Token + "\n      labels: ['other:host']\n"
	if err := os.WriteFile(configured.config, []byte(connection), 0o644); err != nil {
		test.Fatal(err)
	}
	configured.start(test)
	eventually(test, "the configured runner to declare its labels", func() bool {
		runner := runners()["e2e-configured"]
		return runner.Online && strings.Join(runner.Labels, ",") == "other" && runner.Version == runnerVersion
	})

	if status, _ := call(test, http.MethodGet, srv.url+"/api/runners", bearer(limited), ""); status != http.StatusForbidden {
		test.Fatalf("runner list with a token for one repo: HTTP %d", status)
	}
	if status, _ := call(test, http.MethodPost, srv.url+"/api/actions/runner.v1.RunnerService/FetchTask", map[string]string{"Content-Type": "application/json"}, "{}"); status != http.StatusUnauthorized {
		test.Fatalf("FetchTask without runner headers: HTTP %d", status)
	}
	var saved struct {
		UUID  string `json:"uuid"`
		Token string `json:"token"`
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(second.config), "registration.json"))
	if err != nil || json.Unmarshal(data, &saved) != nil || saved.UUID == "" {
		test.Fatalf("registration file: %v: %s", err, data)
	}
	fetch := func(token string) int {
		test.Helper()
		status, _ := call(test, http.MethodPost, srv.url+"/api/actions/runner.v1.RunnerService/FetchTask", map[string]string{
			"Content-Type": "application/json", "x-runner-uuid": saved.UUID, "x-runner-token": token,
		}, "{}")
		return status
	}
	if status := fetch(saved.Token); status != http.StatusOK {
		test.Fatalf("FetchTask with the runner's token: HTTP %d", status)
	}
	if status := fetch(strings.Repeat("0", len(saved.Token))); status != http.StatusUnauthorized {
		test.Fatalf("FetchTask with a wrong token: HTTP %d", status)
	}
}
