package git

import (
	"cmp"
	"fmt"
	"regexp"
	"strconv"
)

func parseVersionTag(ref string) *VersionTag {

	// Define a regex to capture the version part, assuming it starts after the last slash
	// and consists of digits and dots.
	// 2.5.0^{}
	re := regexp.MustCompile(`^(?:.*/)?(\d+)\.(\d+)\.(\d+)(\{.*)?$`)
	if matches := re.FindStringSubmatch(ref); len(matches) > 1 {
		vt := VersionTag{peeled: matches[4] != ""}
		vt.Major, _ = strconv.Atoi(matches[1])
		vt.Minor, _ = strconv.Atoi(matches[2])
		vt.Patch, _ = strconv.Atoi(matches[3])
		return &vt
	}
	return nil
}

type VersionTag struct {
	Major, Minor, Patch int
	SHA                 string
	peeled              bool
}

func (v VersionTag) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// compareVersionTags compares two versionTag structs.
// It prioritizes Major, then Minor, then Patch.
// If all version numbers are equal, it compares based on the peeled field:
// - If both are peeled or both are not peeled, they are considered equal (0).
// - If t1 is not peeled and t2 is peeled, t1 is considered lesser (-1).
// - If t1 is peeled and t2 is not peeled, t1 is considered greater (1).
func compareVersionTags(t1, t2 VersionTag) int {
	if c := cmp.Compare(t1.Major, t2.Major); c != 0 {
		return c
	}
	if c := cmp.Compare(t1.Minor, t2.Minor); c != 0 {
		return c
	}
	if c := cmp.Compare(t1.Patch, t2.Patch); c != 0 {
		return c
	}

	// Logic for peeled comparison (as per your modified requirements)
	// A peeled tag is considered "greater" than a non-peeled tag if versions are equal.
	if t1.peeled == t2.peeled {
		return 0 // Both have same peeled status
	}
	// At this point, t1.peeled != t2.peeled
	if t1.peeled {
		return 1 // t1 is peeled, t2 is not peeled: t1 is greater
	}
	return -1 // t1 is not peeled, t2 is peeled: t1 is lesser
}
