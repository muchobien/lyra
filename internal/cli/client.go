package cli

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

var ErrTimeout = errors.New("timeout")

type client struct {
	api       *lyraapi.Client
	streaming *lyraapi.Client
}

func newClient() *client {
	baseURL := strings.TrimRight(os.Getenv("LYRA_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	token := os.Getenv("LYRA_TOKEN")
	agent := os.Getenv("LYRA_AGENT")
	session := os.Getenv("LYRA_SESSION")
	headers := lyraapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
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
	return &client{api: api, streaming: streaming}
}

func splitRepo(name string) (string, string) {
	owner, repo, _ := strings.Cut(name, "/")
	return owner, repo
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
