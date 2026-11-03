package app

import (
	"encoding/json"
	"errors"
	"log"
	"path/filepath"
	"time"

	"github.com/darylcecile/patch/internal/config"
	"github.com/darylcecile/patch/internal/workspace"
)

type snapshot struct {
	state    []byte
	settings config.Config
}

// One writer owns disk I/O. Snapshots cross the goroutine boundary, never live models.
type persistence struct {
	queue chan snapshot
	stop  chan struct{}
	done  chan struct{}
	errs  chan error
}

func newPersistence(home string, invalidate func()) *persistence {
	p := &persistence{queue: make(chan snapshot, 1), stop: make(chan struct{}), done: make(chan struct{}), errs: make(chan error, 1)}
	go p.run(home, invalidate)
	return p
}

func (p *persistence) enqueue(state *workspace.State, settings *config.Config) error {
	var data []byte
	if state != nil {
		var err error
		data, err = json.Marshal(state)
		if err != nil {
			return err
		}
	}
	select {
	case <-p.queue:
	default:
	}
	p.queue <- snapshot{state: data, settings: *settings}
	return nil
}

func (p *persistence) close() { close(p.stop); <-p.done }

func (p *persistence) run(home string, invalidate func()) {
	defer close(p.done)
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	var pending *snapshot
	write := func() {
		if pending == nil {
			return
		}
		err := pending.settings.Save(filepath.Join(home, "config.json"))
		if len(pending.state) > 0 {
			var state workspace.State
			stateErr := json.Unmarshal(pending.state, &state)
			if stateErr == nil {
				stateErr = state.Save(filepath.Join(home, "session.json"))
			}
			err = errors.Join(err, stateErr)
		}
		pending = nil
		if err != nil {
			log.Printf("patch: save: %v", err)
			select {
			case p.errs <- err:
			default:
			}
			invalidate()
		}
	}
	for {
		select {
		case s := <-p.queue:
			pending = &s
			timer.Reset(250 * time.Millisecond)
		case <-timer.C:
			write()
		case <-p.stop:
			select {
			case s := <-p.queue:
				pending = &s
			default:
			}
			write()
			return
		}
	}
}
