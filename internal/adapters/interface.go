package adapters

import "github.com/mohammedsamin/mcpup/internal/planner"

// Adapter is the contract every client adapter must implement.
type Adapter interface {
	Name() string
	Detect(workspace string) (string, error)
	Read(path string) (planner.ClientState, error)
	Apply(current planner.ClientState, desired planner.ClientState) (planner.Plan, error)
	Write(path string, desired planner.ClientState) error
	Validate(path string) error
}
