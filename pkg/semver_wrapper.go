package pkg

import (
	semverv3 "github.com/Masterminds/semver/v3"
)

type SemverManager interface {
	Compare(v *semverv3.Version) int
}

type semverv3Constraints semverv3.Constraints

// Since this works with constraints, we only know if the version satisfies the constraint or not
// if satisfies, we return 2 to complement semver.Compare's returns
//
//	-1 - a is less than b
//	 0 - a is equal to b
//	 1 - a is breater than b
//	 2 - a satisfies b
//	 3 - a does not satisfy b
func (c semverv3Constraints) Compare(v *semverv3.Version) int {
	_c := semverv3.Constraints(c)

	if _c.Check(v) {
		return 2
	}

	return 3
}
