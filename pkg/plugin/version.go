package plugin

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	errInvalidStage   = errors.New("invalid version stage")
	errInvalidVersion = errors.New("version number must be positive")
	errEmptyPlugin    = errors.New("plugin version is empty")
)

// Stage represents the stability of a version
type Stage uint8

// Order Stage in decreasing degree of stability for comparison purposes.
// StableStage must be 0 so that it is the default Stage
const ( // The order in this const declaration will be used to order version stages except for StableStage
	// StableStage should be used for plugins that are rarely changed in backwards-compatible ways, e.g. bug fixes.
	StableStage Stage = iota
	// BetaStage should be used for plugins that may be changed in minor ways and are not expected to break between uses.
	BetaStage Stage = iota
	// AlphaStage should be used for plugins that are frequently changed and may break between uses.
	AlphaStage Stage = iota
)

const (
	alphaStage  = "alpha"
	betaStage   = "beta"
	stableStage = ""
)

// ParseStage parses stage into a Stage, assuming it is one of the valid stages
func ParseStage(stage string) (Stage, error) {
	var s Stage
	return s, s.Parse(stage)
}

// Parse parses stage inline, assuming it is one of the valid stages
func (s *Stage) Parse(stage string) error {
	switch stage {
	case alphaStage:
		*s = AlphaStage
	case betaStage:
		*s = BetaStage
	case stableStage:
		*s = StableStage
	default:
		return errInvalidVersion
	}
	return nil
}

// String returns the string representation of s
func (s Stage) String() string {
	switch s {
	case AlphaStage:
		return alphaStage
	case BetaStage:
		return betaStage
	case StableStage:
		return stableStage
	default:
		panic(errInvalidStage)
	}
}

// Validate ensures that the stage is one of the valid stages
func (s Stage) Validate() error {
	switch s {
	case AlphaStage:
	case BetaStage:
	case StableStage:
	default:
		return errInvalidStage
	}

	return nil
}

// Compare returns -1 if s < other, 0 if s == other, and 1 if s > other.
func (s Stage) Compare(other Stage) int {
	if s == other {
		return 0
	}

	// Stage are sorted in decreasing order
	if s > other {
		return -1
	}
	return 1
}

// Version is a plugin version containing a non-zero positive integer and a stage value that represents stability.
type Version struct {
	// Number denotes the current version of a plugin. Two different numbers between versions
	// indicate that they are incompatible.
	Number int64
	// Stage indicates stability.
	Stage Stage
}

// ParseVersion parses version into a Version, assuming it adheres to format: (v)?[1-9][0-9]*(-(alpha|beta))?
func ParseVersion(version string) (Version, error) {
	var v Version
	return v, v.Parse(version)
}

// Parse parses version inline, assuming it adheres to format: (v)?[1-9][0-9]*(-(alpha|beta))?
func (v *Version) Parse(version string) (err error) {
	version = strings.TrimPrefix(version, "v")
	if len(version) == 0 {
		return errEmptyPlugin
	}

	substrings := strings.SplitN(version, "-", 2)

	if v.Number, err = strconv.ParseInt(substrings[0], 10, 64); err != nil {
		return
	} else if v.Number < 1 {
		return errInvalidVersion
	}

	if len(substrings) == 1 {
		v.Stage = StableStage
	} else {
		err = v.Stage.Parse(substrings[1])
	}
	return
}

// String returns the string representation of v
func (v Version) String() string {
	if len(v.Stage.String()) == 0 {
		return fmt.Sprintf("v%d", v.Number)
	}
	return fmt.Sprintf("v%d-%s", v.Number, v.Stage)
}

// Validate ensures that the version number is positive and the stage is one of the valid stages
func (v Version) Validate() error {
	if v.Number < 0 {
		return errInvalidVersion
	}
	return v.Stage.Validate()
}

// Compare returns -1 if v < other, 0 if v == other, and 1 if v > other.
func (v Version) Compare(other Version) int {
	if v.Number > other.Number {
		return 1
	} else if v.Number < other.Number {
		return -1
	}

	return v.Stage.Compare(other.Stage)
}

// IsStable returns true if v is stable.
func (v Version) IsStable() bool {
	return v.Stage.Compare(StableStage) == -1
}
