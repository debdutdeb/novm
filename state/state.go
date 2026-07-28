package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/debdutdeb/novm/v3/common"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

type version = string

type State struct {
	Update updateState `json:"update"`

	PoolControl struct {
		Usage          map[version]lastHitState `json:"usage"`
		LastControlled time.Time                `json:"lastControlled,omitempty"`
	} `json:"poolControl"`
}

type updateState struct {
	LastChecked  time.Time `json:"lastChecked"`
	TimesChecked int       `json:"timesChecked"`
}

type lastHitState struct {
	Hits           int       `json:"hits,omitempty"`
	LastUsed       time.Time `json:"lastUsed,omitempty"`
	FirstInstalled time.Time `json:"firstInstalled,omitempty"`
}

func NewState() (*State, error) {
	root := common.RootDir

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

	if state.PoolControl.Usage == nil {
		state.PoolControl.Usage = make(map[version]lastHitState)
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

func (s *State) Save() error {
	root := common.RootDir

	f, err := os.OpenFile(filepath.Join(root, "state.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0750)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")

	return encoder.Encode(s)
}

func (s *State) IncUpdateCheck() error {
	if time.Since(s.Update.LastChecked) >= time.Hour {
		s.Update.TimesChecked = 1
	} else {
		s.Update.TimesChecked++
	}

	s.Update.LastChecked = time.Now()

	return s.Save()
}

func (s *State) IncPoolHit(v version) error {
	control := s.PoolControl.Usage[v]
	control.Hits++
	control.LastUsed = time.Now()
	if control.FirstInstalled.Equal(time.Time{}) {
		control.FirstInstalled = time.Now()
	}
	s.PoolControl.Usage[v] = control
	return s.Save()
}

func (l *lastHitState) hasItBeen10DaysSinceLastUsed() bool {
	return time.Since(l.LastUsed) > time.Hour*24*10
}

func (l *lastHitState) hasUsageAveragedOver10TimesPer3Days() bool {
	var threeDaysSinceInstalled = max(1, time.Since(l.FirstInstalled)/24/3)
	return l.Hits/int(threeDaysSinceInstalled) > 10
}

func (s *State) ShouldClearPoolCache(v version) bool {
	control, exists := s.PoolControl.Usage[v]
	if exists && control.hasItBeen10DaysSinceLastUsed() && !control.hasUsageAveragedOver10TimesPer3Days() {
		return true
	}
	return false
}

func (s *State) ShouldControl() bool {
	return time.Since(s.PoolControl.LastControlled) >= time.Hour*24
}

var compactionlockfile = common.InRootDir(".versions.compact.lock")

func (s *State) acquirePoolCompactLock() error {
	fd, err := unix.Open(compactionlockfile, unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC, 0600)

	if err != nil {
		if os.IsExist(err) {
			st := unix.Stat_t{}
			if err := unix.Stat(compactionlockfile, &st); err != nil {
				return fmt.Errorf("failed to process exising lock file: %w", err)
			}
			if time.Since(time.Unix(st.Mtim.Sec, 0)) > time.Hour*24 {
				if err := s.releasePoolCompactLock(); err != nil {
					return fmt.Errorf("failed to release 24 hours old stale lock file: %w", err)
				}

				return s.acquirePoolCompactLock()
			}
		}
		return err
	}

	return unix.Close(fd)
}

func (s *State) releasePoolCompactLock() error {
	return os.Remove(compactionlockfile)
}

func (s *State) WhileCompactingPool(fn func(s *State) error, filter func(v version) bool) error {
	if !s.ShouldControl() {
		return fn(s)
	}

	if err := s.acquirePoolCompactLock(); err != nil {
		if os.IsExist(err) {
			return fn(s)
		}

		return err
	}

	/*
		Multiple instances can fight and end up dropping a version that's currently in use; even with a lock
		This is a `wontfix` for now. It will add significant overhead to the compaction logic, each instance will need to reliably communicate back its usage.
		even leases will have race conditions that won't be 100% bug free. so for now, wontfix.
	*/

	errg := errgroup.Group{}
	errg.SetLimit(-1)

	defer s.releasePoolCompactLock()

	versions, err := common.ListVersions()
	if err != nil {
		return errors.Join(errg.Wait(), err)
	}
	for _, ver := range versions {
		if filter(ver) && s.ShouldClearPoolCache(ver) {
			errg.Go(func() error { return os.RemoveAll(common.Where(ver)) })
		}
	}

	err2 := fn(s)

	// after a process we simply let go, as another process likely already finished with its cpompaction

	if err := errg.Wait(); err != nil {
		// if compaction fails, we don't update lastControlled
		return errors.Join(err, err2)
	}

	s.PoolControl.LastControlled = time.Now()
	return errors.Join(s.Save(), err2)
}
