package judge

type SelectionJob struct {
	SubmissionID      int    `db:"SUBMISSION_ID"`
	Solution          string `db:"SOLUTION"`
	ReferenceSolution string `db:"REFERENCE_SOLUTION"`
	SchemaName        string `db:"SCHEMA_NAME"`
	CheckOrder        string `db:"CHECK_ORDER"`
}

type DataManipulationJob struct {
	SubmissionID      int    `db:"SUBMISSION_ID"`
	Solution          string `db:"SOLUTION"`
	ReferenceSolution string `db:"REFERENCE_SOLUTION"`
}

type SchemaManipulationJob struct {
	SubmissionID      int    `db:"SUBMISSION_ID"`
	Solution          string `db:"SOLUTION"`
	ReferenceSolution string `db:"REFERENCE_SOLUTION"`
}
