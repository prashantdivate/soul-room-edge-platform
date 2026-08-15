package ota

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type downloader struct {
	cfg    Config
	client *http.Client
}

func newDownloader(cfg Config) downloader {
	return downloader{cfg: cfg, client: &http.Client{Timeout: cfg.DownloadTimeout, CheckRedirect: func(req *http.Request, _ []*http.Request) error {
		if req.URL.Scheme != "https" {
			return errors.New("artifact redirect must remain on HTTPS")
		}
		return nil
	}}}
}

func (d downloader) fetch(ctx context.Context, request Request) (string, error) {
	if !strings.HasPrefix(strings.ToLower(request.ArtifactURL), "https://") {
		return "", errors.New("artifact URL must use HTTPS")
	}
	final := filepath.Join(d.cfg.StagingDir, safeID(request.UpdateID)+".artifact")
	partial := final + ".part"
	if digestMatches(final, request.Digest) {
		return final, nil
	}

	var offset int64
	if info, err := os.Stat(partial); err == nil {
		offset = info.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, request.ArtifactURL, nil)
	if err != nil {
		return "", err
	}
	if offset > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(offset, 10)+"-")
	}
	response, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("artifact server returned %s", response.Status)
	}
	if response.Request.URL.Scheme != "https" {
		return "", errors.New("artifact download left HTTPS")
	}
	if response.ContentLength > 0 && d.cfg.MaxArtifactBytes > 0 && offset+response.ContentLength > d.cfg.MaxArtifactBytes {
		return "", errors.New("artifact server reported a payload larger than the configured limit")
	}
	if offset > 0 && response.StatusCode == http.StatusPartialContent && !strings.HasPrefix(response.Header.Get("Content-Range"), "bytes "+strconv.FormatInt(offset, 10)+"-") {
		return "", errors.New("artifact server returned an invalid resume range")
	}
	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 && response.StatusCode == http.StatusPartialContent {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
		offset = 0
	}
	f, err := os.OpenFile(partial, flags, 0600)
	if err != nil {
		return "", err
	}
	limit := d.cfg.MaxArtifactBytes
	if limit <= 0 {
		limit = 4 << 30
	}
	written, copyErr := io.Copy(f, io.LimitReader(response.Body, limit-offset+1))
	syncErr := f.Sync()
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if syncErr != nil {
		return "", syncErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if offset+written > limit {
		return "", errors.New("download exceeded the configured artifact size limit")
	}
	if request.ArtifactSize > 0 && offset+written != request.ArtifactSize {
		return "", fmt.Errorf("artifact size mismatch: expected %d bytes, received %d", request.ArtifactSize, offset+written)
	}
	if !digestMatches(partial, request.Digest) {
		_ = os.Remove(partial)
		return "", errors.New("artifact SHA-256 digest mismatch")
	}
	if err := os.Rename(partial, final); err != nil {
		return "", err
	}
	return final, nil
}

func digestMatches(path, expected string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == strings.ToLower(expected)
}
