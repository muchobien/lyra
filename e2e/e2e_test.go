package e2e

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

const login = "tester@example.com"

var lyraBinary string

type server struct {
	url     string
	address string
	data    string
	stop    func()
}

type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (locked *lockedBuffer) Write(data []byte) (int, error) {
	locked.mutex.Lock()
	defer locked.mutex.Unlock()
	return locked.buffer.Write(data)
}

func (locked *lockedBuffer) String() string {
	locked.mutex.Lock()
	defer locked.mutex.Unlock()
	return locked.buffer.String()
}

type cliResult struct {
	stdout string
	stderr string
	code   int
}

func TestMain(suite *testing.M) {
	dir, err := os.MkdirTemp("", "lyra-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lyraBinary = filepath.Join(dir, "lyra")
	build := exec.Command("go", "build", "-o", lyraBinary, "../cmd/lyra")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build lyra:", err)
		os.Exit(1)
	}
	code := suite.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func startServer(test *testing.T, trustedProxy string) server {
	test.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		test.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	return startServerAt(test, address, filepath.Join(test.TempDir(), "data"), trustedProxy)
}

func startServerAt(test *testing.T, address, data, trustedProxy string) server {
	test.Helper()
	cmd := exec.Command(lyraBinary, "serve", "--listen", address, "--data", data, "--trusted-proxy", trustedProxy, "--allow-login", login)
	logs := &lockedBuffer{}
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err := cmd.Start(); err != nil {
		test.Fatal(err)
	}
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cmd.Process.Kill()
			cmd.Wait()
		})
	}
	test.Cleanup(func() {
		stop()
		if test.Failed() {
			test.Logf("server %s log:\n%s", address, logs.String())
		}
	})
	url := "http://" + address
	for attempt := 0; attempt < 100; attempt++ {
		response, err := http.Get(url + "/healthz")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return server{url: url, address: address, data: data, stop: stop}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	test.Fatalf("server did not start:\n%s", logs.String())
	return server{}
}

func call(test *testing.T, method, url string, headers map[string]string, body string) (int, []byte) {
	test.Helper()
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		test.Fatal(err)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		test.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		test.Fatal(err)
	}
	return response.StatusCode, data
}

func tailnet() map[string]string {
	return map[string]string{"Tailscale-User-Login": login}
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func createToken(test *testing.T, srv server, body string) string {
	test.Helper()
	status, data := call(test, http.MethodPost, srv.url+"/api/tokens", tailnet(), body)
	if status != http.StatusCreated {
		test.Fatalf("create token: HTTP %d: %s", status, data)
	}
	var created struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(data, &created); err != nil {
		test.Fatal(err)
	}
	return created.Secret
}

func adminToken(test *testing.T, srv server) string {
	return createToken(test, srv, `{"name":"admin","scope":"admin"}`)
}

func startLyra(test *testing.T, srv server, token string, args ...string) func() cliResult {
	test.Helper()
	cmd := exec.Command(lyraBinary, args...)
	cmd.Env = append(os.Environ(), "LYRA_URL="+srv.url, "LYRA_TOKEN="+token, "LYRA_AGENT=e2e-agent", "LYRA_SESSION=session-123")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		test.Fatal(err)
	}
	return func() cliResult {
		test.Helper()
		err := cmd.Wait()
		code := 0
		if exitError, isExit := err.(*exec.ExitError); isExit {
			code = exitError.ExitCode()
		} else if err != nil {
			test.Fatal(err)
		}
		return cliResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
	}
}

func runLyra(test *testing.T, srv server, token string, args ...string) cliResult {
	test.Helper()
	return startLyra(test, srv, token, args...)()
}

func lyraJSON(test *testing.T, srv server, token string, target any, args ...string) {
	test.Helper()
	result := runLyra(test, srv, token, args...)
	if result.code != 0 {
		test.Fatalf("lyra %s: exit %d: %s", strings.Join(args, " "), result.code, result.stderr)
	}
	if err := json.Unmarshal([]byte(result.stdout), target); err != nil {
		test.Fatalf("lyra %s: %v: %s", strings.Join(args, " "), err, result.stdout)
	}
}

func gitCommand(dir string, stdin []byte, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GIT_MERGE_AUTOEDIT=no",
		"GIT_AUTHOR_NAME=Test Agent", "GIT_AUTHOR_EMAIL=agent@example.com",
		"GIT_COMMITTER_NAME=Test Agent", "GIT_COMMITTER_EMAIL=agent@example.com",
	)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func mustGit(test *testing.T, dir string, args ...string) string {
	test.Helper()
	output, err := gitCommand(dir, nil, args...)
	if err != nil {
		test.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return output
}

func newSource(test *testing.T) string {
	test.Helper()
	dir := filepath.Join(test.TempDir(), "src")
	mustGit(test, "", "init", "-q", "-b", "main", dir)
	return dir
}

func commitFile(test *testing.T, dir, name, content, message string, extra ...string) {
	test.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		test.Fatal(err)
	}
	mustGit(test, dir, "add", name)
	mustGit(test, dir, append(extra, "commit", "-q", "-m", message)...)
}

func signedWith(key string) []string {
	return []string{"-c", "gpg.format=ssh", "-c", "user.signingkey=" + key}
}

func newKey(test *testing.T, name string) string {
	test.Helper()
	path := filepath.Join(test.TempDir(), name)
	if output, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", name, "-f", path).CombinedOutput(); err != nil {
		test.Fatalf("ssh-keygen: %v: %s", err, output)
	}
	return path
}

func fingerprint(test *testing.T, publicKey string) string {
	test.Helper()
	output, err := exec.Command("ssh-keygen", "-l", "-f", publicKey).Output()
	if err != nil {
		test.Fatal(err)
	}
	return strings.Fields(string(output))[1]
}

func createRepo(test *testing.T, srv server, token, name string) {
	test.Helper()
	var repo store.Repo
	lyraJSON(test, srv, token, &repo, "repo", "create", name)
}

func TestAuth(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	untrusted := startServer(test, "10.9.9.9")

	if status, _ := call(test, http.MethodGet, srv.url+"/api/me", nil, ""); status != http.StatusUnauthorized {
		test.Errorf("no credentials: HTTP %d, want 401", status)
	}

	headers := tailnet()
	headers["Lyra-Agent"] = "curl"
	status, data := call(test, http.MethodGet, srv.url+"/api/me", headers, "")
	var me struct {
		Actor store.Actor `json:"actor"`
		Scope string      `json:"scope"`
	}
	if err := json.Unmarshal(data, &me); err != nil || status != http.StatusOK {
		test.Fatalf("tailnet me: HTTP %d: %s", status, data)
	}
	if me.Actor != (store.Actor{User: login, Via: "tailscale", Agent: "curl"}) || me.Scope != "admin" {
		test.Errorf("tailnet me = %+v scope %s", me.Actor, me.Scope)
	}

	if status, _ := call(test, http.MethodGet, srv.url+"/api/me", map[string]string{"Tailscale-User-Login": "intruder@example.com"}, ""); status != http.StatusForbidden {
		test.Errorf("login not allowed: HTTP %d, want 403", status)
	}
	if status, _ := call(test, http.MethodGet, untrusted.url+"/api/me", tailnet(), ""); status != http.StatusUnauthorized {
		test.Errorf("header from untrusted address: HTTP %d, want 401", status)
	}

	token := adminToken(test, srv)
	headers = bearer(token)
	headers["Lyra-Agent"] = "e2e-agent"
	headers["Lyra-Session"] = "session-123"
	status, data = call(test, http.MethodGet, srv.url+"/api/me", headers, "")
	if err := json.Unmarshal(data, &me); err != nil || status != http.StatusOK {
		test.Fatalf("token me: HTTP %d: %s", status, data)
	}
	if me.Actor != (store.Actor{User: login, Via: "token:admin", Agent: "e2e-agent", Session: "session-123"}) {
		test.Errorf("token me = %+v", me.Actor)
	}

	request, _ := http.NewRequest(http.MethodGet, srv.url+"/api/me", nil)
	request.SetBasicAuth("anyone", token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		test.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		test.Errorf("basic auth with token: HTTP %d, want 200", response.StatusCode)
	}
}

func TestCLI(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)

	var created struct {
		store.Repo
		DefaultBranch string      `json:"default_branch"`
		Branches      []repos.Ref `json:"branches"`
		CloneURL      string      `json:"clone_url"`
	}
	lyraJSON(test, srv, token, &created, "repo", "create", "demo", "--description", "Demo repo")
	if created.Name != "demo" || created.Description != "Demo repo" || created.DefaultBranch != "main" || created.CloneURL != srv.url+"/demo.git" || len(created.Branches) != 0 || created.CreatedBy != login {
		test.Errorf("repo create = %+v", created)
	}

	if result := runLyra(test, srv, token, "repo", "create", "bad/name"); result.code != 1 || !strings.Contains(result.stderr, `"error":"invalid repository name`) {
		test.Errorf("invalid name: exit %d stderr %s", result.code, result.stderr)
	}
	if result := runLyra(test, srv, token, "repo", "get"); result.code != 2 {
		test.Errorf("missing argument: exit %d, want 2", result.code)
	}
	if result := runLyra(test, srv, token, "repo", "list", "--bogus"); result.code != 2 {
		test.Errorf("unknown flag: exit %d, want 2", result.code)
	}
	if result := runLyra(test, srv, token, "repo", "list", "--jq", ".[].name"); result.stdout != "demo\n" {
		test.Errorf("--jq output %q", result.stdout)
	}
	if result := runLyra(test, srv, token, "me", "--jq", ".actor.via"); result.stdout != "token:admin\n" {
		test.Errorf("me via %q", result.stdout)
	}

	if result := runLyra(test, srv, token, "repo", "delete", "demo"); result.code != 2 {
		test.Errorf("delete without --yes: exit %d, want 2", result.code)
	}
	if result := runLyra(test, srv, token, "repo", "delete", "demo", "--yes"); result.code != 0 {
		test.Fatalf("delete: exit %d: %s", result.code, result.stderr)
	}
	if _, err := os.Stat(filepath.Join(srv.data, "repos", "demo.git")); !os.IsNotExist(err) {
		test.Errorf("demo.git still on disk: %v", err)
	}
}

func TestGitTransport(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "demo")
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "First commit")

	auth := "http.extraHeader=Authorization: Bearer " + token
	mustGit(test, src, "-c", auth, "-c", "http.extraHeader=Lyra-Agent: e2e-agent", "-c", "http.extraHeader=Lyra-Session: session-123", "push", "-q", srv.url+"/demo.git", "main")

	blob := make([]byte, 3_000_000)
	rand.Read(blob)
	commitFile(test, src, "blob.bin", string(blob), "Add random blob")
	mustGit(test, src, "-c", auth, "-c", "http.postBuffer=1024", "push", "-q", srv.url+"/demo.git", "main")

	if output, err := gitCommand("", nil, "clone", "-q", srv.url+"/demo.git", filepath.Join(test.TempDir(), "anonymous")); err == nil {
		test.Errorf("clone without credentials succeeded: %s", output)
	}

	clone := filepath.Join(test.TempDir(), "clone")
	helper := "credential.helper=!f() { echo username=lyra; echo password=" + token + "; }; f"
	mustGit(test, "", "-c", "credential.helper=", "-c", helper, "clone", "-q", srv.url+"/demo", clone)
	if got, want := mustGit(test, clone, "rev-parse", "HEAD"), mustGit(test, src, "rev-parse", "HEAD"); got != want {
		test.Errorf("clone HEAD %s, want %s", got, want)
	}

	tailnetClone := filepath.Join(test.TempDir(), "tailnet")
	header := "http.extraHeader=Tailscale-User-Login: " + login
	mustGit(test, "", "-c", header, "clone", "-q", srv.url+"/demo.git", tailnetClone)
	mustGit(test, tailnetClone, "commit", "-q", "--allow-empty", "-m", "Pushed over the tailnet header")
	mustGit(test, tailnetClone, "-c", header, "-c", "http.extraHeader=Lyra-Agent: tailnet-agent", "push", "-q", "origin", "main")

	var events []store.Event
	lyraJSON(test, srv, token, &events, "event", "list", "--repo", "demo")
	pushes := []store.Actor{}
	for _, event := range events {
		if event.Action == "push" {
			pushes = append(pushes, event.Actor)
		}
	}
	want := []store.Actor{
		{User: login, Via: "tailscale", Agent: "tailnet-agent"},
		{User: login, Via: "token:admin"},
		{User: login, Via: "token:admin", Agent: "e2e-agent", Session: "session-123"},
	}
	if fmt.Sprint(pushes) != fmt.Sprint(want) {
		test.Errorf("push actors = %+v, want %+v", pushes, want)
	}
}

func TestTokens(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	admin := adminToken(test, srv)
	createRepo(test, srv, admin, "demo")
	createRepo(test, srv, admin, "other")
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+admin, "push", "-q", srv.url+"/demo.git", "main")

	var created struct {
		Token  store.Token `json:"token"`
		Secret string      `json:"secret"`
	}
	lyraJSON(test, srv, admin, &created, "token", "create", "--name", "ci-read", "--scope", "read", "--repo", "demo", "--expires", "1h")
	read := created.Secret
	if created.Token.Scope != "read" || fmt.Sprint(created.Token.Repos) != "[demo]" || created.Token.ExpiresAt == nil || !strings.HasPrefix(read, "lyra_") {
		test.Errorf("token create = %+v", created.Token)
	}

	if result := runLyra(test, srv, read, "repo", "list", "--jq", "[.[].name] | join(\",\")"); result.stdout != "demo\n" {
		test.Errorf("read token repo list %q", result.stdout)
	}
	if status, _ := call(test, http.MethodGet, srv.url+"/api/repos/other", bearer(read), ""); status != http.StatusForbidden {
		test.Errorf("read token on other repo: HTTP %d, want 403", status)
	}
	readHeader := "http.extraHeader=Authorization: Bearer " + read
	clone := filepath.Join(test.TempDir(), "clone")
	mustGit(test, "", "-c", readHeader, "clone", "-q", srv.url+"/demo.git", clone)
	mustGit(test, clone, "commit", "-q", "--allow-empty", "-m", "Rejected")
	if output, err := gitCommand(clone, nil, "-c", readHeader, "push", "-q", "origin", "main"); err == nil || !strings.Contains(output, "403") {
		test.Errorf("read token push: err %v output %s", err, output)
	}
	if result := runLyra(test, srv, read, "repo", "create", "sneaky"); result.code != 1 || !strings.Contains(result.stderr, "lacks admin") {
		test.Errorf("read token repo create: exit %d stderr %s", result.code, result.stderr)
	}

	listing := runLyra(test, srv, admin, "token", "list")
	if strings.Contains(listing.stdout, "hash") || strings.Contains(listing.stdout, read) {
		test.Errorf("token list exposes the hash or secret")
	}
	var tokens []store.Token
	if err := json.Unmarshal([]byte(listing.stdout), &tokens); err != nil || len(tokens) != 2 {
		test.Fatalf("token list: %v: %s", err, listing.stdout)
	}
	if tokens[1].LastUsedAt == nil {
		test.Errorf("read token last_used_at not recorded")
	}

	if result := runLyra(test, srv, admin, "token", "revoke", fmt.Sprint(created.Token.ID)); result.code != 0 {
		test.Fatalf("revoke: %s", result.stderr)
	}
	if status, _ := call(test, http.MethodGet, srv.url+"/api/me", bearer(read), ""); status != http.StatusUnauthorized {
		test.Errorf("revoked token: HTTP %d, want 401", status)
	}

	short := createToken(test, srv, `{"name":"short","scope":"read","expires_in":"1s"}`)
	if status, _ := call(test, http.MethodGet, srv.url+"/api/me", bearer(short), ""); status != http.StatusOK {
		test.Errorf("fresh token: HTTP %d, want 200", status)
	}
	time.Sleep(1500 * time.Millisecond)
	if status, _ := call(test, http.MethodGet, srv.url+"/api/me", bearer(short), ""); status != http.StatusUnauthorized {
		test.Errorf("expired token: HTTP %d, want 401", status)
	}
}

func TestSignatures(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "demo")
	agentKey := newKey(test, "agent")
	strangerKey := newKey(test, "stranger")
	src := newSource(test)

	commitFile(test, src, "README.md", "# Demo\n", "Signed by the agent key", append(signedWith(agentKey), "-c", "commit.gpgsign=true")...)
	commitFile(test, src, "README.md", "# Demo\nunsigned\n", "Unsigned commit")
	commitFile(test, src, "README.md", "# Demo\nstranger\n", "Signed by an unregistered key", append(signedWith(strangerKey), "-c", "commit.gpgsign=true")...)
	mustGit(test, src, append(signedWith(agentKey), "tag", "-s", "v0.1.0", "-m", "First tag")...)
	mustGit(test, src, "tag", "-a", "v0.0.1", "-m", "Unsigned annotated tag", "HEAD~2")
	mustGit(test, src, "tag", "lightweight", "HEAD~1")
	mustGit(test, src, "checkout", "-q", "-b", "side")
	commitFile(test, src, "side.txt", "side\n", "Side branch commit")
	mustGit(test, src, append(signedWith(agentKey), "tag", "-s", "side-v1", "-m", "Signed side tag")...)
	mustGit(test, src, "checkout", "-q", "main")
	mustGit(test, src, append(signedWith(agentKey), "merge", "-q", "--no-ff", "-S", "-m", "Signed merge of a signed tag", "side-v1")...)
	if !strings.Contains(mustGit(test, src, "cat-file", "commit", "HEAD"), "\nmergetag ") {
		test.Fatal("merge commit has no mergetag header")
	}

	signed := mustGit(test, src, "log", "--format=%H", "--grep=^Signed by the agent key$", "main")
	raw := mustGit(test, src, "cat-file", "commit", signed) + "\n"
	tampered, err := gitCommand(src, []byte(strings.Replace(raw, "Signed by the agent key", "Tampered message", 1)), "hash-object", "-t", "commit", "-w", "--stdin")
	if err != nil {
		test.Fatal(tampered)
	}
	unsigned := mustGit(test, src, "log", "--format=%H", "--grep=^Unsigned commit$", "main")
	payload := mustGit(test, src, "cat-file", "commit", unsigned) + "\n"
	payloadFile := filepath.Join(test.TempDir(), "payload")
	if err := os.WriteFile(payloadFile, []byte(payload), 0o644); err != nil {
		test.Fatal(err)
	}
	if output, err := exec.Command("ssh-keygen", "-q", "-Y", "sign", "-n", "file", "-f", agentKey, payloadFile).CombinedOutput(); err != nil {
		test.Fatalf("ssh-keygen sign: %v: %s", err, output)
	}
	fileSignature, err := os.ReadFile(payloadFile + ".sig")
	if err != nil {
		test.Fatal(err)
	}
	headerEnd := strings.Index(payload, "\n\n")
	forged := payload[:headerEnd+1] + "gpgsig " + strings.ReplaceAll(strings.TrimSuffix(string(fileSignature), "\n"), "\n", "\n ") + "\n" + payload[headerEnd+1:]
	wrongNamespace, err := gitCommand(src, []byte(forged), "hash-object", "-t", "commit", "-w", "--stdin")
	if err != nil {
		test.Fatal(wrongNamespace)
	}

	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/demo.git",
		"main", "--tags", tampered+":refs/heads/tampered", wrongNamespace+":refs/heads/wrong-namespace")

	commitStatus := func(rev string) repos.Signature {
		test.Helper()
		var commit repos.Commit
		lyraJSON(test, srv, token, &commit, "commit", "show", "demo", rev)
		return commit.Signature
	}
	if status := commitStatus(signed).Status; status != repos.Unverified {
		test.Errorf("signed commit before its key is registered: %s, want unverified", status)
	}

	agentPublic := agentKey + ".pub"
	var key store.Key
	lyraJSON(test, srv, token, &key, "key", "add", "--signing", "--agent", "claude", "--title", "e2e agent key", agentPublic)
	if key.Fingerprint != fingerprint(test, agentPublic) || key.Agent != "claude" || !key.Signing {
		test.Errorf("key add = %+v", key)
	}
	if result := runLyra(test, srv, token, "key", "add", "--signing", agentKey); result.code != 2 || !strings.Contains(result.stderr, "private key") {
		test.Errorf("private key add: exit %d stderr %s", result.code, result.stderr)
	}

	var commits []repos.Commit
	lyraJSON(test, srv, token, &commits, "commit", "list", "demo")
	got := map[string]repos.Signature{}
	for _, commit := range commits {
		got[commit.Subject] = commit.Signature
	}
	agentSignature := repos.Signature{Status: repos.Verified, Fingerprint: key.Fingerprint, KeyID: key.ID, KeyUser: login, KeyAgent: "claude"}
	wantCommits := map[string]repos.Signature{
		"Signed merge of a signed tag":  agentSignature,
		"Side branch commit":            {Status: repos.Unsigned},
		"Signed by an unregistered key": {Status: repos.Unverified, Fingerprint: fingerprint(test, strangerKey+".pub")},
		"Unsigned commit":               {Status: repos.Unsigned},
		"Signed by the agent key":       agentSignature,
	}
	for subject, want := range wantCommits {
		if fmt.Sprintf("%+v", got[subject]) != fmt.Sprintf("%+v", want) {
			test.Errorf("commit %q signature = %+v, want %+v", subject, got[subject], want)
		}
	}

	var tags []repos.Tag
	lyraJSON(test, srv, token, &tags, "tag", "list", "demo")
	wantTags := map[string]string{"v0.1.0": repos.Verified, "side-v1": repos.Verified, "v0.0.1": repos.Unsigned, "lightweight": repos.Unsigned}
	for _, tag := range tags {
		if tag.Signature.Status != wantTags[tag.Name] {
			test.Errorf("tag %s = %s, want %s", tag.Name, tag.Signature.Status, wantTags[tag.Name])
		}
	}

	for name, rev := range map[string]string{"tampered": tampered, "wrong namespace": wrongNamespace} {
		if signature := commitStatus(rev); signature.Status != repos.Unverified {
			test.Errorf("%s commit = %s, want unverified", name, signature.Status)
		}
	}

	bare := filepath.Join(srv.data, "repos", "demo.git")
	allowedSigners := filepath.Join(test.TempDir(), "allowed_signers")
	publicKey, err := os.ReadFile(agentPublic)
	if err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(allowedSigners, []byte(login+` namespaces="git" `+string(publicKey)), 0o644); err != nil {
		test.Fatal(err)
	}
	gitStatus := func(verify, kind, object string) string {
		if _, err := gitCommand(bare, nil, "-c", "gpg.ssh.allowedSignersFile="+allowedSigners, verify, object); err == nil {
			return repos.Verified
		}
		objectType := mustGit(test, bare, "cat-file", "-t", object)
		content := mustGit(test, bare, "cat-file", objectType, object)
		if (kind == "commit" && strings.Contains(content, "\ngpgsig ")) || (kind == "tag" && objectType == "tag" && strings.Contains(content, "-----BEGIN SSH SIGNATURE-----")) {
			return repos.Unverified
		}
		return repos.Unsigned
	}
	for _, sha := range strings.Fields(mustGit(test, bare, "rev-list", "--all")) {
		if lyra, git := commitStatus(sha).Status, gitStatus("verify-commit", "commit", sha); lyra != git {
			test.Errorf("commit %s: lyra %s, git verify-commit %s", sha, lyra, git)
		}
	}
	for _, tag := range tags {
		if git := gitStatus("verify-tag", "tag", tag.Name); tag.Signature.Status != git {
			test.Errorf("tag %s: lyra %s, git verify-tag %s", tag.Name, tag.Signature.Status, git)
		}
	}

	if result := runLyra(test, srv, token, "key", "delete", fmt.Sprint(key.ID)); result.code != 0 {
		test.Fatalf("key delete: %s", result.stderr)
	}
	if result := runLyra(test, srv, token, "commit", "list", "demo", "--jq", `[.[] | select(.signature.status == "verified")] | length`); result.stdout != "0\n" {
		test.Errorf("verified commits after deleting the key: %s", result.stdout)
	}
}

func TestEvents(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "demo")
	createRepo(test, srv, token, "other")
	auth := "http.extraHeader=Authorization: Bearer " + token
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	latest := func() int64 {
		test.Helper()
		var events []store.Event
		lyraJSON(test, srv, token, &events, "event", "list", "--limit", "1")
		return events[0].ID
	}

	cursor := fmt.Sprint(latest())
	waiting := startLyra(test, srv, token, "event", "wait", "--repo", "demo", "--action", "push", "--since", cursor, "--timeout", "30s")
	mustGit(test, src, "-c", auth, "push", "-q", srv.url+"/other.git", "main")
	createRepo(test, srv, token, "third")
	mustGit(test, src, "-c", auth, "push", "-q", srv.url+"/demo.git", "main")
	result := waiting()
	var pushed store.Event
	if err := json.Unmarshal([]byte(result.stdout), &pushed); err != nil || result.code != 0 {
		test.Fatalf("wait for push: exit %d: %v: %s %s", result.code, err, result.stdout, result.stderr)
	}
	if pushed.Action != "push" || pushed.Repo != "demo" || pushed.Actor.Via != "token:admin" || !strings.Contains(string(pushed.Data), "refs/heads/main") {
		test.Errorf("wait for push = %+v %s", pushed, pushed.Data)
	}

	var again store.Event
	lyraJSON(test, srv, token, &again, "event", "wait", "--repo", "demo", "--action", "push", "--since", cursor, "--timeout", "5s")
	if again.ID != pushed.ID {
		test.Errorf("wait --since after the push returned event %d, want %d", again.ID, pushed.ID)
	}
	if result := runLyra(test, srv, token, "event", "wait", "--since", cursor, "--action", "repo.*", "--timeout", "5s", "--jq", ".repo"); result.stdout != "third\n" {
		test.Errorf("wait for repo.* = %q %s", result.stdout, result.stderr)
	}
	if result := runLyra(test, srv, token, "event", "wait", "--action", "repo.*", "--timeout", "1s"); result.code != 3 || strings.TrimSpace(result.stderr) != `{"error":"timeout"}` {
		test.Errorf("wait without --since matched an earlier event: exit %d stdout %s stderr %s", result.code, result.stdout, result.stderr)
	}

	scoped := createToken(test, srv, `{"name":"other-only","scope":"read","repos":["other"]}`)
	if result := runLyra(test, srv, scoped, "event", "wait", "--repo", "demo", "--timeout", "5s"); result.code != 1 || !strings.Contains(result.stderr, "HTTP 403") {
		test.Errorf("scoped token waiting on demo: exit %d stderr %s", result.code, result.stderr)
	}
	if result := runLyra(test, srv, scoped, "event", "wait", "--since", "0", "--timeout", "5s", "--jq", `[.action, .repo] | join(" ")`); result.stdout != "repo.create other\n" {
		test.Errorf("scoped token first visible event = %q %s", result.stdout, result.stderr)
	}
	if result := runLyra(test, srv, token, "event", "wait", "--since", "-1"); result.code != 2 {
		test.Errorf("negative --since: exit %d, want 2", result.code)
	}
	if result := runLyra(test, srv, token, "event", "wait", "--action", "[", "--timeout", "5s"); result.code != 1 || !strings.Contains(result.stderr, "invalid action pattern") {
		test.Errorf("bad pattern: exit %d stderr %s", result.code, result.stderr)
	}

	before := latest()
	waiting = startLyra(test, srv, token, "event", "wait", "--repo", "demo", "--action", "push", "--timeout", "30s")
	time.Sleep(time.Second)
	srv.stop()
	restarted := startServerAt(test, srv.address, srv.data, "127.0.0.1")
	commitFile(test, src, "second.md", "second\n", "Second commit")
	mustGit(test, src, "-c", auth, "push", "-q", restarted.url+"/demo.git", "main")
	result = waiting()
	if err := json.Unmarshal([]byte(result.stdout), &pushed); err != nil || result.code != 0 || pushed.Action != "push" || pushed.ID <= before {
		test.Errorf("wait across a restart: exit %d: %s %s", result.code, result.stdout, result.stderr)
	}
}
