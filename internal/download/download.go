// Package download fetches data files from GitHub repositories, pinned to
// one commit so that an index always records exactly which revision of
// the data it was built from. Both data sources (STEPBible and the UBS
// open-license collection) are fetched this way.
package download

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// ErrHTTP is returned when GitHub answers with an error status.
var ErrHTTP = errors.New("falha no download")

// LatestCommit asks GitHub for the current commit of a repository's
// default branch ("owner/name").
func LatestCommit(ctx context.Context, client *http.Client, repo string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/repos/"+repo+"/commits/HEAD", nil)
	if err != nil {
		return "", fmt.Errorf("download: building request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: asking GitHub for the latest commit of %s: %w", repo, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: %w: GitHub respondeu HTTP %d (%s)", ErrHTTP, resp.StatusCode, repo)
	}

	var body struct {
		SHA string `json:"sha"`
	}

	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return "", fmt.Errorf("download: reading commit: %w", err)
	}

	return body.SHA, nil
}

// RawURL is the address of a file of repo at commit; path may contain
// spaces ("Translators Amalgamated OT+NT/...").
func RawURL(repo, commit, path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}

	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s",
		repo, commit, strings.Join(segments, "/"))
}

// Fetch downloads rawURL into dest, through a temporary file renamed at
// the end, so an interrupted download never leaves a truncated file
// under the final name.
func Fetch(ctx context.Context, client *http.Client, rawURL, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("download: building request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download: %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: %w: HTTP %d em %s", ErrHTTP, resp.StatusCode, rawURL)
	}

	tmp := dest + ".part"

	out, err := os.Create(tmp) //nolint:gosec // tmp is under the program's own data directory
	if err != nil {
		return fmt.Errorf("download: creating %s: %w", tmp, err)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = out.Close()

		return fmt.Errorf("download: writing %s: %w", tmp, err)
	}

	err = out.Close()
	if err != nil {
		return fmt.Errorf("download: closing %s: %w", tmp, err)
	}

	err = os.Rename(tmp, dest)
	if err != nil {
		return fmt.Errorf("download: renaming %s: %w", tmp, err)
	}

	return nil
}
