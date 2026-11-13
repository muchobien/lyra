package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrTimeout = errors.New("timeout")

type client struct {
	baseURL   string
	token     string
	agent     string
	session   string
	http      *http.Client
	streaming *http.Client
}

func newClient() *client {
	baseURL := strings.TrimRight(os.Getenv("LYRA_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &client{
		baseURL:   baseURL,
		token:     os.Getenv("LYRA_TOKEN"),
		agent:     os.Getenv("LYRA_AGENT"),
		session:   os.Getenv("LYRA_SESSION"),
		http:      &http.Client{Timeout: 60 * time.Second},
		streaming: http.DefaultClient,
	}
}

func (client *client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body, escaped)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if client.token != "" {
		req.Header.Set("Authorization", "Bearer "+client.token)
	}
	if client.agent != "" {
		req.Header.Set("Lyra-Agent", client.agent)
	}
	if client.session != "" {
		req.Header.Set("Lyra-Session", client.session)
	}
	return req, nil
}

func (client *client) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	req, err := client.newRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	response, err := client.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, responseError(response)
	}
	return io.ReadAll(response.Body)
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

func (client *client) waitForEvent(ctx context.Context, path string, timeout time.Duration) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	lastID := ""
	for {
		req, err := client.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "text/event-stream")
		if lastID != "" {
			req.Header.Set("Last-Event-ID", lastID)
		}
		response, err := client.streaming.Do(req)
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

func (client *client) uploadAsset(ctx context.Context, path, file string, clobber bool) ([]byte, error) {
	handle, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	info, err := handle.Stat()
	if err != nil {
		return nil, err
	}
	query := url.Values{"name": {filepath.Base(file)}}
	if clobber {
		query.Set("clobber", "true")
	}
	req, err := client.newRequest(ctx, http.MethodPost, path+"/assets?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Body = handle
	req.ContentLength = info.Size()
	contentType := mime.TypeByExtension(filepath.Ext(file))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	response, err := client.streaming.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, responseError(response)
	}
	return io.ReadAll(response.Body)
}

func (client *client) downloadAsset(ctx context.Context, path, file, sha string) error {
	req, err := client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	req.Header.Del("Accept")
	response, err := client.streaming.Do(req)
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
	if got := hex.EncodeToString(hash.Sum(nil)); got != sha {
		return fmt.Errorf("sha256 is %s, expected %s", got, sha)
	}
	return os.Rename(temporary.Name(), file)
}
