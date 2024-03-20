package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/debdutdeb/node-proxy/common"
)

type State struct {
	Update updateState `json:"update"`
}

type updateState struct {
	LastChecked  time.Time `json:"lastChecked"`
	TimesChecked int       `json:"timesChecked"`
}

func NewState() (*State, error) {
	root, err := common.RootDir()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filepath.Join(root, "state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}

		return nil, err
	}

	var state State

	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, err
	}

	return &state, f.Close()
}

func (s *State) ShouldCheckForUpdate() bool {
	if s.Update.TimesChecked == 60 {
		// we should never reach this
		return false
	}

	if time.Since(s.Update.LastChecked) < time.Minute {
		return false
	}

	return true
}

func (s *State) IncUpdateCheck() error {
	if time.Since(s.Update.LastChecked) >= time.Hour {
		s.Update.TimesChecked = 1
	} else {
		s.Update.TimesChecked++
	}

	s.Update.LastChecked = time.Now()

	root, err := common.RootDir()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(root, "state.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0750)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")

	return encoder.Encode(s)
}
