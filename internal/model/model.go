package model

type Repository struct {
	Name        string
	Slug        string
	Private     bool
	Description string
	ProjectName string
	CloneURL    string
}

type RepoStatus string

const (
	StatusSuccess RepoStatus = "success"
	StatusFailed  RepoStatus = "failed"
	StatusSkipped RepoStatus = "skipped"
)

type RepoResult struct {
	Repo   Repository
	Status RepoStatus
	Reason string
}
