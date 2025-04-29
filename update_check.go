package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/mod/semver"

	gopark "github.com/debdutdeb/gopark/pkg/utils"
	"github.com/debdutdeb/node-proxy/common"
	st "github.com/debdutdeb/node-proxy/state"
	"github.com/debdutdeb/node-proxy/versions"
)

type releasesResponse struct {
	Tag    string  `json:"tag_name"`
	Assets []asset `json:"assets"`
}

type asset struct {
	Name string `json:"name"`
	Url  string `json:"browser_download_url"`
}

func wrapInUpdateCheck(action func() error) error {
	var wg sync.WaitGroup

	wg.Add(1)

	var actionErr error

	// let the action run
	go func() {
		actionErr = action()
		wg.Done()
	}()

	updateErr := checkUpdate(&wg)

	wg.Wait()

	if actionErr != nil && updateErr != nil {
		return fmt.Errorf("actionerr: %w, updatecheckerr: %w", actionErr, updateErr)
	}

	if actionErr != nil {
		return actionErr
	}

	return updateErr
}

func checkUpdate(wg *sync.WaitGroup) error {
	var (
		err         error
		req         *http.Request
		resp        *http.Response
		release     releasesResponse
		tmpDownload string

		state *st.State
	)

	waitAndLog := func(msg string, args ...any) {
		wg.Wait()
		log.Printf(msg, args...)
	}

	state, err = st.NewState()
	if err != nil {
		waitAndLog("[ERROR] failed to load current state: %v", err)
		return err
	}

	if !state.ShouldCheckForUpdate() {
		return nil
	}

	err = state.IncUpdateCheck()
	if err != nil {
		waitAndLog("[ERROR] failed to update state: %v", err)
		return err
	}

	req, err = http.NewRequest("GET", "https://api.github.com/repos/debdutdeb/novm/releases/latest", nil)
	if err != nil {
		waitAndLog("[ERROR] failed to fetch latest update: %v", err)
		return err
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	resp, err = (&http.Client{}).Do(req)
	if err != nil {
		waitAndLog("[ERROR] failed to fetch latest update: %v", err)
		return err
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		waitAndLog("[ERROR] failed to fetch latest update: %v", err)
		return err
	}

	if semver.Compare(versions.Version, release.Tag) != -1 {
		waitAndLog("no new novm updates found.")
		return nil
	}

	var dir string

	dir, err = gopark.MkdirTemp("", common.BIN_NAME)
	if err != nil {
		waitAndLog("[ERROR] failed to download latest binary: %v", err)
		return err
	}

	tmpDownload = filepath.Join(dir, common.BIN_NAME)

	for _, asset := range release.Assets {
		if asset.Name == common.BIN_NAME+"-"+runtime.GOOS+"-"+runtime.GOARCH {
			if err := gopark.DownloadSilent(asset.Url, tmpDownload); err != nil {
				waitAndLog("[ERROR] failed to download latest binary: %v", err)
				return err
			}

			break
		}
	}

	// we ignore sigint and sigterm here to not lose the binary

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		for {
			<-sig

			go waitAndLog("ignoring signal since novm is still updating")
		}
	}()

	// Only update the binary once action is complete

	wg.Wait()

	// we expect current binary to be a symlink

	log.Printf("Updating novm to %s", release.Tag)

	bin, err := currentExecutable()
	if err != nil {
		log.Fatalf("failed to get novm binary to upgrade: %v", err)
		return err
	}

	if err := os.Rename(tmpDownload, bin); err != nil {
		log.Fatalf("failed to move download to bin path: %v", err)
		return err
	}

	return nil
}

// TODO aggregate maybe
func currentExecutable() (string, error) {
	path, err1 := os.Executable()
	if err1 == nil {
		return filepath.EvalSymlinks(path)
	}

	path, err2 := exec.LookPath("node")
	if err2 == nil {
		return filepath.EvalSymlinks(path)
	}

	return "", fmt.Errorf("%w, %w", err1, err2)
}
