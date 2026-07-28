package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/debdutdeb/novm/v3/common"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

type version = string

type State struct {
	lockfd int

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
	for ver, control := range s.PoolControl.Usage {
		if ver == v && control.hasItBeen10DaysSinceLastUsed() && !control.hasUsageAveragedOver10TimesPer3Days() {
			return true
		}
	}
	return false
}

func (s *State) ShouldControl() bool {
	return time.Since(s.PoolControl.LastControlled) >= time.Hour*24
}

func (s *State) acquirePoolCompactLock() error {
	fd, err := unix.Open(common.InRootDir(".versions.compact.lock"), unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC, 0600)

	if err != nil && !os.IsExist(err) {
		return err
	}

	defer unix.Close(fd)

	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		return err
	}

	s.lockfd = fd

	return nil
}

func (s *State) releasePoolCompactLock() error {
	err := syscall.Flock(s.lockfd, syscall.LOCK_UN)
	return err
}

func (s *State) WhileCompactingPool(fn func(s *State) error, filter func(v version) bool) error {
	if !s.ShouldControl() {
		return fn(s)
	}

	/*
		Multiple instances can fight and end up dropping a version that's currently in use; even with a lock
	*/

	errch := make(chan error, 1)

	errg := errgroup.Group{}
	errg.SetLimit(-1)

	go func() {
		// this will lock goroutine if another process is still running;
		if err := s.acquirePoolCompactLock(); err != nil {
			if os.IsExist(err) {
				close(errch)
				return
			}

			errch <- err
			return
		}

		defer s.releasePoolCompactLock()

		if !s.ShouldControl() {
			return
		}

		versions, err := common.ListVersions()
		if err != nil {
			err2 := errg.Wait()
			errch <- errors.Join(err, err2)
			return
		}
		for _, ver := range versions {
			if filter(ver) && s.ShouldClearPoolCache(ver) {
				errg.Go(func() error { return os.RemoveAll(common.Where(ver)) })
			}
		}
	}()

	select {
	case err := <-errch:
		return err
	default:
		err2 := fn(s)

		// after a process we simply let go, as another process likely already finished with its cpompaction

		err3, _ := <-errch

		if err := errg.Wait(); err != nil {
			return errors.Join(err, err2, err3)
		}

		s.PoolControl.LastControlled = time.Now()
		return errors.Join(s.Save(), err2, err3)
	}

}
