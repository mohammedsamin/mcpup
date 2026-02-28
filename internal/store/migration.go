package store

import "fmt"

// IsSupportedSchemaVersion reports whether mcpup can directly load a schema version.
func IsSupportedSchemaVersion(version int) bool {
	return version == CurrentSchemaVersion
}

// PlannedMigrationPath returns the expected migration path for a source version.
// v1 policy:
// - Current version is loaded directly.
// - Older versions require explicit migrators when introduced.
// - Newer versions are rejected to avoid unsafe writes.
func PlannedMigrationPath(fromVersion int) ([]int, error) {
	switch {
	case fromVersion <= 0:
		return nil, fmt.Errorf("invalid schema version %d", fromVersion)
	case fromVersion == CurrentSchemaVersion:
		return []int{CurrentSchemaVersion}, nil
	case fromVersion < CurrentSchemaVersion:
		return nil, fmt.Errorf("no migrator registered from version %d to %d", fromVersion, CurrentSchemaVersion)
	default:
		return nil, fmt.Errorf("config version %d is newer than supported version %d", fromVersion, CurrentSchemaVersion)
	}
}
