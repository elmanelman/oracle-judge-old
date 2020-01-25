package judge

const (
	Unknown int = iota
	PendingReview
	OnReview
	Accepted
	ExecutionError
	RestrictionViolated
	IncorrectContent
	IncorrectOrder
)

type Verdict struct {
	SubmissionID       int    `db:"SUBMISSION_ID"`
	SubmissionStatusID int    `db:"SUBMISSION_STATUS_ID"`
	ReviewerMessage    string `db:"REVIEWER_MESSAGE"`
}
