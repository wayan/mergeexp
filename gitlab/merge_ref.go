package gitlab

import "fmt"

type mergeRef struct {
	*MergeRequest
}

func (m mergeRef) Name() string {
	return fmt.Sprintf("MR %d: %s", m.MergeRequest.ID, m.MergeRequest.Title)
}

func (m mergeRef) Sha() string {
	return m.MergeRequest.Sha
}

func (mr *MergeRequest) MergeRef() mergeRef {
	return mergeRef{MergeRequest: mr}
}
