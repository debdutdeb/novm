package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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

func startCheckUpdate() (chan bool, chan bool) {
	cont := make(chan bool)
	done := make(chan bool)

	// don't run on novm call
	if filepath.Base(os.Args[0]) == common.BIN_NAME {
		// comsume the channel
		go func() {
			<-cont
			done <- true
		}()

		return cont, done
	}

	go func() {
		defer func() {
			done <- true
		}()

		var (
			err         error
			req         *http.Request
			resp        *http.Response
			release     releasesResponse
			tmpDownload string

			state *st.State
		)

		state, err = st.NewState()
		if err != nil {
			<-cont
			log.Printf("[ERROR] failed to load current state: %v", err)
			return
		}

		if !state.ShouldCheckForUpdate() {
			<-cont
			return
		}

		err = state.IncUpdateCheck()
		if err != nil {
			<-cont
			log.Printf("[ERROR] failed to update state: %v", err)
			return
		}

		req, err = http.NewRequest("GET", "https://api.github.com/repos/debdutdeb/novm/releases/latest", nil)
		if err != nil {
			<-cont
			log.Printf("[ERROR] failed to fetch latest update: %v", err)
			return
		}

		req.Header.Add("Accept", "application/vnd.github+json")
		req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

		resp, err = (&http.Client{}).Do(req)
		if err != nil {
			<-cont
			log.Printf("[ERROR] failed to fetch latest update: %v", err)
			return
		}

		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			<-cont
			log.Printf("[ERROR] failed to fetch latest update: %v", err)
			return
		}

		if semver.Compare(versions.Version, release.Tag) == -1 {
			var dir string

			dir, err = gopark.MkdirTemp("", common.BIN_NAME)
			if err != nil {
				<-cont
				log.Printf("[ERROR] failed to download latest binary: %v", err)
				return
			}

			tmpDownload = filepath.Join(dir, common.BIN_NAME)

			for _, asset := range release.Assets {
				if asset.Name == common.BIN_NAME+"-"+runtime.GOOS+"-"+runtime.GOARCH {
					if err := gopark.DownloadSilent(asset.Url, tmpDownload); err != nil {
						<-cont
						log.Printf("[ERROR] failed to download latest binary: %v", err)
						return
					}

					break
				}
			}

		} else {
			<-cont // consume
			log.Println("no new novm updates found.")
			return
		}

		// we ignore sigint and sigterm here to not lose the binary

		sig := make(chan os.Signal, 1)

		signal.Notify(sig, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

		go func() {
			for {
				<-sig

				log.Println("ignoring signal since novm is still updating")
			}
		}()

		<-cont // upgrade the binary

		// we expect current binary to be a symlink

		log.Printf("Updating novm to %s", release.Tag)

		root, _ := common.RootDir()

		bin := filepath.Join(root, "bin", "node")

		novmBin, err := filepath.EvalSymlinks(bin)
		if err != nil {
			log.Fatalf("failed to get novm binary to upgrade: %v", err)
		}

		src, err := os.Open(tmpDownload)
		if err != nil {
			log.Fatalf("failed to updated binary (%s): %v", tmpDownload, err)
		}

		dst, err := os.OpenFile(novmBin, os.O_WRONLY|os.O_TRUNC, 0750)
		if err != nil {
			log.Fatalf("failed to updated binary (%s): %v", novmBin, err)
		}

		io.Copy(dst, src)

		os.Remove(bin)

		if err := os.Link(novmBin, bin); err != nil {
			log.Fatalf("failed to install novm: %v\ntry running `novm install` manually", err)
		}
	}()

	return cont, done
}
