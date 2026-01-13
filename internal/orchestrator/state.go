package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	StateFileName = ".rlm_state.json"
)

// SaveState persists the orchestrator state to disk
func (o *Orchestrator) SaveState() error {
	state := State{
		Stack:       o.stack,
		CurrentTask: o.currentTask,
		Results:     o.results,
		Stats:       o.stats,
		Timestamp:   time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	stateFile := filepath.Join(o.config.WorkDir, StateFileName)
	return os.WriteFile(stateFile, data, 0644)
}

// LoadState restores the orchestrator state from disk
func (o *Orchestrator) LoadState() error {
	stateFile := filepath.Join(o.config.WorkDir, StateFileName)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state file is fine (fresh start)
		}
		return err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	o.stack = state.Stack
	o.currentTask = state.CurrentTask
	o.results = state.Results
	o.stats = state.Stats

	o.logger.Info().
		Int("stack_depth", len(o.stack)).
		Int("results", len(o.results)).
		Msg("Restored state from disk")

	return nil
}

// ClearState removes the state file
func (o *Orchestrator) ClearState() error {
	stateFile := filepath.Join(o.config.WorkDir, StateFileName)
	err := os.Remove(stateFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// HasState checks if a state file exists
func (o *Orchestrator) HasState() bool {
	stateFile := filepath.Join(o.config.WorkDir, StateFileName)
	_, err := os.Stat(stateFile)
	return err == nil
}
