package gitlab

// minimal info about merge request
type MergeRequest struct {
	ID              int    `json:"id"`
	Sha             string `json:"sha"`
	SourceProjectId int    `json:"source_project_id"`
	TargetProjectId int    `json:"target_project_id"`
	TargetBranch    string `json:"target_branch"`
	Title           string `json:"title"`
}
