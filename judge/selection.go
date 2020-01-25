package judge

import (
	"fmt"
	"github.com/elmanelman/oracle-judge/config"
	"github.com/elmanelman/oracle-judge/templates"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"
)

type SelectionJudge struct {
	logger    *zap.Logger
	mainDB    *sqlx.DB
	waitGroup *sync.WaitGroup
	stop      chan struct{}
	verdicts  chan Verdict

	selectionDBs map[string]*sqlx.DB
	fetchTicker  *time.Ticker
	jobs         chan SelectionJob
}

func NewSelectionJudge(
	logger *zap.Logger,
	mainDB *sqlx.DB,
	waitGroup *sync.WaitGroup,
	stop chan struct{},
	verdicts chan Verdict,
) *SelectionJudge {
	return &SelectionJudge{
		logger:       logger,
		mainDB:       mainDB,
		waitGroup:    waitGroup,
		stop:         stop,
		verdicts:     verdicts,
		selectionDBs: map[string]*sqlx.DB{},
		jobs:         make(chan SelectionJob),
	}
}

func (j *SelectionJudge) Start(cfg config.SelectionJudgeConfig) error {
	if err := j.ConnectSelectionDBs(cfg); err != nil {
		return err
	}
	j.fetchTicker = time.NewTicker(time.Duration(cfg.FetchPeriod) * time.Millisecond)

	j.waitGroup.Add(1 + cfg.ReviewerCount)

	go j.StartFetching()
	for id := 1; id <= cfg.ReviewerCount; id++ {
		go j.SelectionReviewer(id)
	}

	return nil
}

func (j *SelectionJudge) Stop() {
	j.fetchTicker.Stop()
}

func (j *SelectionJudge) ConnectSelectionDBs(cfg config.SelectionJudgeConfig) error {
	for _, c := range cfg.DBConfigs {
		connectionString := c.Config.ConnectionString()

		db, err := connectDB(connectionString)
		if err != nil {
			return err
		}

		j.selectionDBs[c.Name] = db

		j.logger.Info(
			"selection DB connected",
			zap.String("connection_string", connectionString),
		)
	}

	return nil
}

func (j *SelectionJudge) FetchJobs() error {
	query := templates.FetchSelectionJobs
	rows, err := j.mainDB.Queryx(query)
	if err != nil {
		return err
	}
	if rows == nil {
		return nil
	}

	var job SelectionJob
	for rows.Next() {
		if err = rows.StructScan(&job); err != nil {
			return err
		}

		j.verdicts <- Verdict{
			SubmissionID:       job.SubmissionID,
			SubmissionStatusID: OnReview,
		}

		j.jobs <- job
	}

	return nil
}

func (j *SelectionJudge) StartFetching() {
	defer func() {
		j.logger.Info("stopped fetching selection jobs")
		j.waitGroup.Done()
	}()
	for {
		select {
		case <-j.stop:
			return
		case <-j.fetchTicker.C:
			if err := j.FetchJobs(); err != nil {
				j.logger.Error("failed fetching selection jobs")
			}
		}
	}
}

func (j *SelectionJudge) SelectionReviewer(reviewerID int) {
	defer func() {
		j.logger.Info(
			"stopped selection reviewer",
			zap.Int("reviewer_id", reviewerID),
		)
		j.waitGroup.Done()
	}()
	for {
		select {
		case <-j.stop:
			return
		case job := <-j.jobs:
			job.Solution = prepareSelectionSolution(job.Solution)
			job.ReferenceSolution = prepareSelectionSolution(job.ReferenceSolution)

			// check if review schema exists
			if j.selectionDBs[job.SchemaName] == nil {
				j.logger.Error(
					"selection DB does not exist",
					zap.String("database_name", job.SchemaName),
				)
				continue
			}

			// fetch task restrictions
			restrictions, err := j.FetchRestrictions(job.SubmissionID)
			if err != nil {
				j.logger.Error(
					"failed to fetch task restrictions",
					zap.Int("submission_id", job.SubmissionID),
					zap.String("error_message", err.Error()),
				)
				continue
			}

			// check task restrictions
			violated := false
			for _, r := range restrictions {
				if strings.Contains(job.Solution, strings.ToUpper(r)) {
					j.verdicts <- Verdict{
						SubmissionID:       job.SubmissionID,
						SubmissionStatusID: RestrictionViolated,
						ReviewerMessage:    fmt.Sprintf("\"%s\" is restricted", r),
					}
					violated = true
					break
				}
			}
			if violated {
				continue
			}

			// review selection solution result content
			isContentCorrect, err := j.reviewSelectionContent(job)
			if err != nil {
				j.logger.Error(
					"error reviewing selection content",
					zap.Int("submission_id", job.SubmissionID),
					zap.String("error_message", err.Error()),
				)
				j.verdicts <- Verdict{
					SubmissionID:       job.SubmissionID,
					SubmissionStatusID: ExecutionError,
					ReviewerMessage:    err.Error(),
				}
				continue
			}
			if !isContentCorrect {
				j.verdicts <- Verdict{
					SubmissionID:       job.SubmissionID,
					SubmissionStatusID: IncorrectContent,
				}
				continue
			}

			// review selection solution result order (if needed)
			if job.CheckOrder == "N" {
				j.verdicts <- Verdict{
					SubmissionID:       job.SubmissionID,
					SubmissionStatusID: Accepted,
				}
				continue
			}

			isOrderCorrect, err := j.reviewSelectionOrder(job)
			if err != nil {
				j.logger.Error(
					"error reviewing selection order",
					zap.Int("submission_id", job.SubmissionID),
					zap.String("error_message", err.Error()),
				)
				j.verdicts <- Verdict{
					SubmissionID:       job.SubmissionID,
					SubmissionStatusID: ExecutionError,
					ReviewerMessage:    err.Error(),
				}
				continue
			}
			if !isOrderCorrect {
				j.verdicts <- Verdict{
					SubmissionID:       job.SubmissionID,
					SubmissionStatusID: IncorrectOrder,
				}
				continue
			}

			j.logger.Info("WTF")

			// accept solution
			j.verdicts <- Verdict{
				SubmissionID:       job.SubmissionID,
				SubmissionStatusID: Accepted,
			}
		}
	}
}

func (j *SelectionJudge) reviewSelection(job SelectionJob, checkOrder bool) (bool, error) {
	var queryTemplate string
	if checkOrder {
		queryTemplate = templates.ReviewSelectionOrder
	} else {
		queryTemplate = templates.ReviewSelectionContent
	}
	query := fmt.Sprintf(queryTemplate, job.Solution, job.ReferenceSolution)
	rows, err := j.selectionDBs[job.SchemaName].Query(query)
	if err != nil {
		return false, err
	}
	if rows == nil || !rows.Next() {
		return true, nil
	}
	return false, nil
}

func (j *SelectionJudge) reviewSelectionContent(job SelectionJob) (bool, error) {
	return j.reviewSelection(job, false)
}

func (j *SelectionJudge) reviewSelectionOrder(job SelectionJob) (bool, error) {
	return j.reviewSelection(job, true)
}
