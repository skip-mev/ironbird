package messages

type CreateGitHubCheckRequest struct {
	InstallationID int64
	Owner          string
	Repo           string
	Name           string
	SHA            string
	Status         *string
	Conclusion     *string
	Title          *string
	Summary        *string
}

type CreateGitHubCheckResponse int64

type UpdateGitHubCheckRequest struct {
	CheckID        int64
	InstallationID int64
	Owner          string
	Repo           string
	Name           string
	Status         *string
	Conclusion     *string
	Title          *string
	Summary        *string
	Text           string
}

type UpdateGitHubCheckResponse int64
