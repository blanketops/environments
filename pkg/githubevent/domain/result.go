package domain

type GitHubEventResult struct {
	Accepted     bool
	Triggered    bool
	Message      string
	TriggeredRef string // optional: Build name, Deployment, etc
}

// ToStatus converts a GitHubEventResult to a GitHubEventStatus.
func (r GitHubEventResult) ToStatus() GitHubEventStatus {
	return GitHubEventStatus(r)
}

func Accepted(msg string) GitHubEventResult {
	return GitHubEventResult{
		Accepted: true,
		Message:  msg,
	}
}

func Triggered(ref, msg string) GitHubEventResult {
	return GitHubEventResult{
		Accepted:     true,
		Triggered:    true,
		TriggeredRef: ref,
		Message:      msg,
	}
}

func Rejected(msg string) GitHubEventResult {
	return GitHubEventResult{
		Accepted: false,
		Message:  msg,
	}
}
