package adapters

import "mcpup/internal/planner"

// HarnessResult captures before/after planning and write behavior for adapter tests.
type HarnessResult struct {
	Before planner.ClientState
	Plan   planner.Plan
	After  planner.ClientState
}

// RunHarness executes read->apply->(optional write)->validate for adapter tests.
func RunHarness(adapter Adapter, path string, desired planner.ClientState, dryRun bool) (HarnessResult, error) {
	before, err := adapter.Read(path)
	if err != nil {
		return HarnessResult{}, err
	}

	plan, err := adapter.Apply(before, desired)
	if err != nil {
		return HarnessResult{}, err
	}

	result := HarnessResult{
		Before: before,
		Plan:   plan,
		After:  before,
	}

	if dryRun || !plan.HasChanges() {
		return result, nil
	}

	if err := adapter.Write(path, desired); err != nil {
		return HarnessResult{}, err
	}
	if err := adapter.Validate(path); err != nil {
		return HarnessResult{}, err
	}

	after, err := adapter.Read(path)
	if err != nil {
		return HarnessResult{}, err
	}
	result.After = after
	return result, nil
}
