package merger

import "github.com/wayan/mergeexp/gitdir"

type Merger struct {
	dir             *gitdir.Dir
	ConflictRetries int
}

func New(dir *gitdir.Dir) *Merger {
	return &Merger{
		dir:             dir,
		ConflictRetries: 3,
	}
}
