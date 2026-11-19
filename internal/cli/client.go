package cli

import (
	"bufio"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

var (
	ErrTimeout = errors.New("timeout")
	ErrPending = errors.New("pending")
)

type client struct {
	api       *lyraapi.Client
	streaming *lyraapi.Client
	baseURL   string
}

func newClient() *client {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	configFile := filepath.Join(configDir, "lyra", "config.json")
	var config struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	data, err := os.ReadFile(configFile)
	if err == nil {
		err = json.Unmarshal(data, &config)
	}
	if err == nil && config.Token != "" {
		if info, statErr := os.Stat(configFile); statErr == nil && info.Mode().Perm()&0o077 != 0 {
			err = errors.New("holds a token, so only its owner may read it (chmod 600)")
		}
	}
	var configErr error
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		configErr = fmt.Errorf("%s: %w", configFile, err)
	}
	baseURL := strings.TrimRight(cmp.Or(os.Getenv("LYRA_URL"), config.URL), "/")
	if baseURL == "" && configErr == nil {
		configErr = errors.New("LYRA_URL isn't set; set it or add \"url\" to ~/.config/lyra/config.json")
	}
	token := cmp.Or(os.Getenv("LYRA_TOKEN"), config.Token)
	agent := os.Getenv("LYRA_AGENT")
	if agent == "" && os.Getenv("CLAUDECODE") != "" {
		agent = "claude"
	} else if agent == "" && os.Getenv("CODEX_SESSION_ID") != "" {
		agent = "codex"
	}
	session := cmp.Or(os.Getenv("LYRA_SESSION"), os.Getenv("CLAUDE_CODE_SESSION_ID"), os.Getenv("CODEX_SESSION_ID"))
	headers := lyraapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		if configErr != nil {
			return configErr
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if agent != "" {
			req.Header.Set("Lyra-Agent", agent)
		}
		if session != "" {
			req.Header.Set("Lyra-Session", session)
		}
		return nil
	})
	api, _ := lyraapi.NewClient(baseURL, lyraapi.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}), headers)
	streaming, _ := lyraapi.NewClient(baseURL, lyraapi.WithHTTPClient(http.DefaultClient), headers)
	return &client{api: api, streaming: streaming, baseURL: baseURL}
}

func splitRepo(name string) (string, string) {
	owner, repo, _ := strings.Cut(name, "/")
	return owner, repo
}

func (app *app) repoScoped(commands ...*cobra.Command) []*cobra.Command {
	for _, command := range commands {
		command.Flags().StringP("repo", "R", "", "repository as owner/name (default: $LYRA_REPO, then the git remote)")
		run := command.RunE
		command.RunE = func(cmd *cobra.Command, args []string) error {
			repo, err := app.client.repository(cmd)
			if err != nil {
				return err
			}
			return run(cmd, append([]string{repo}, args...))
		}
	}
	return commands
}

func (client *client) repository(cmd *cobra.Command) (string, error) {
	name, _ := cmd.Flags().GetString("repo")
	name = cmp.Or(name, os.Getenv("LYRA_REPO"))
	if name == "" {
		return client.remoteRepository(cmd)
	}
	owner, repo, found := strings.Cut(name, "/")
	if !found || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", UsageError{fmt.Errorf("--repo %q: want owner/name", name)}
	}
	return name, nil
}

func (client *client) remoteRepository(cmd *cobra.Command) (string, error) {
	server, err := url.Parse(client.baseURL)
	if err != nil || server.Hostname() == "" {
		return "", UsageError{errors.New("pass --repo owner/name")}
	}
	output, err := exec.CommandContext(cmd.Context(), "git", "remote", "-v").Output()
	if err != nil {
		return "", UsageError{errors.New("not in a git repository; pass --repo owner/name")}
	}
	remotes := map[string]string{}
	for line := range strings.Lines(string(output)) {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if repo, found := remoteRepo(fields[1], server.Hostname()); found {
			remotes[fields[0]] = repo
		}
	}
	names := slices.Sorted(maps.Keys(remotes))
	repos := slices.Compact(slices.Sorted(maps.Values(remotes)))
	switch {
	case len(repos) == 1:
		return repos[0], nil
	case remotes["origin"] != "":
		return remotes["origin"], nil
	case len(repos) == 0:
		return "", UsageError{fmt.Errorf("no git remote points at %s; pass --repo owner/name", server.Hostname())}
	}
	return "", UsageError{fmt.Errorf("git remotes %s point at different repositories; pass --repo owner/name",
		strings.Join(names, ", "))}
}

func currentBranch(cmd *cobra.Command) (string, error) {
	output, err := exec.CommandContext(cmd.Context(), "git", "branch", "--show-current").Output()
	branch := strings.TrimSpace(string(output))
	if err != nil || branch == "" {
		return "", UsageError{errors.New("not on a git branch; pass a branch or pull request number")}
	}
	return branch, nil
}

func remoteRepo(remote, host string) (string, bool) {
	remoteHost, path := "", ""
	if parsed, err := url.Parse(remote); err == nil && parsed.Host != "" {
		remoteHost, path = parsed.Hostname(), parsed.Path
	} else if address, scpPath, found := strings.Cut(remote, ":"); found && !strings.Contains(address, "/") {
		remoteHost, path = address[strings.LastIndex(address, "@")+1:], scpPath
	}
	if !strings.EqualFold(remoteHost, host) {
		return "", false
	}
	name := strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	owner, repo, found := strings.Cut(name, "/")
	if !found || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", false
	}
	return name, true
}

func parseNumber(value string) (int64, error) {
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil || number <= 0 {
		return 0, UsageError{fmt.Errorf("%q isn't a number", value)}
	}
	return number, nil
}

func read(response *http.Response, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, responseError(response)
	}
	return io.ReadAll(response.Body)
}

func decode[T any](response *http.Response, err error) (T, []byte, error) {
	var value T
	output, err := read(response, err)
	if err != nil {
		return value, nil, err
	}
	return value, output, json.Unmarshal(output, &value)
}

func (app *app) show(cmd *cobra.Command) func(*http.Response, error) error {
	return func(response *http.Response, err error) error {
		output, err := read(response, err)
		if err != nil {
			return err
		}
		return app.emit(cmd, output)
	}
}

type apiError struct {
	Error  string          `json:"error"`
	Errors []registryError `json:"errors"`
}

type registryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func responseError(response *http.Response) error {
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var failure apiError
	if json.Unmarshal(data, &failure) == nil && failure.Error != "" {
		return fmt.Errorf("%s (HTTP %d)", failure.Error, response.StatusCode)
	}
	if len(failure.Errors) > 0 {
		first := failure.Errors[0]
		return fmt.Errorf("%s: %s (HTTP %d)", first.Code, first.Message, response.StatusCode)
	}
	return fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
}

func (client *client) waitForEvent(
	ctx context.Context,
	params lyraapi.StreamEventsParams,
	timeout time.Duration,
) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	accept := lyraapi.RequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set("Accept", "text/event-stream")
		return nil
	})
	lastID := ""
	for {
		if lastID != "" {
			params.LastEventID = &lastID
		}
		response, err := client.streaming.StreamEvents(ctx, &params, accept)
		if err == nil {
			if response.StatusCode >= 400 && response.StatusCode < 500 {
				defer response.Body.Close()
				return nil, responseError(response)
			}
			var data []byte
			data, lastID = readEvent(response, lastID)
			response.Body.Close()
			if len(data) > 0 {
				return data, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, ErrTimeout
		case <-time.After(time.Second):
		}
	}
}

func readEvent(response *http.Response, lastID string) ([]byte, string) {
	reader := bufio.NewReader(response.Body)
	data := []byte{}
	for response.StatusCode == http.StatusOK {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if value, found := strings.CutPrefix(line, "id:"); found {
			lastID = strings.TrimSpace(value)
		} else if value, found := strings.CutPrefix(line, "data:"); found {
			data = append(data, strings.TrimPrefix(value, " ")...)
		} else if line == "" && len(data) > 0 {
			return data, lastID
		}
	}
	return nil, lastID
}

func (client *client) uploadAsset(
	ctx context.Context,
	repo string,
	id int64,
	file string,
	clobber bool,
) ([]byte, error) {
	handle, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	info, err := handle.Stat()
	if err != nil {
		return nil, err
	}
	contentType := mime.TypeByExtension(filepath.Ext(file))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	length := lyraapi.RequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.ContentLength = info.Size()
		return nil
	})
	owner, name := splitRepo(repo)
	params := lyraapi.UploadAssetParams{Name: filepath.Base(file), Clobber: &clobber}
	return read(client.streaming.UploadAssetWithBody(ctx, owner, name, id, &params, contentType, handle, length))
}

func (client *client) downloadAsset(
	ctx context.Context,
	repo string,
	id int64,
	asset lyraapi.Asset,
	file string,
) error {
	owner, name := splitRepo(repo)
	response, err := client.streaming.DownloadAsset(ctx, owner, name, id, asset.Name)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return responseError(response)
	}
	temporary, err := os.CreateTemp(filepath.Dir(file), ".lyra-download-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	hash := sha256.New()
	_, err = io.Copy(io.MultiWriter(temporary, hash), response.Body)
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != asset.Sha256 {
		return fmt.Errorf("sha256 is %s, expected %s", got, asset.Sha256)
	}
	return os.Rename(temporary.Name(), file)
}
