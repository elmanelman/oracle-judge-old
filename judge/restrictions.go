package judge

import "github.com/elmanelman/oracle-judge/templates"

func (j *SelectionJudge) FetchRestrictions(submissionID int) ([]string, error) {
	query := templates.FetchTaskRestrictions
	var restrictions []string
	err := j.mainDB.Select(&restrictions, query, submissionID)
	if err != nil {
		return nil, err
	}
	return restrictions, nil
}
