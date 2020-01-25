package judge

import (
	"github.com/elmanelman/oracle-judge/config"
	"github.com/elmanelman/oracle-judge/templates"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"sync"
)

type Judges struct {
	logger *zap.Logger

	mainDB *sqlx.DB

	waitGroup *sync.WaitGroup
	stop      chan struct{}

	selectionJudge *SelectionJudge

	//dataManipulationTicker *time.Ticker
	//dataManipulationJobs   chan DataManipulationJob
	//dataManipulationJudge  DataManipulationJudge

	//schemaManipulationTicker *time.Ticker
	//schemaManipulationJobs   chan SchemaManipulationJob
	//schemaManipulationJudge  SchemaManipulationJudge

	verdicts chan Verdict
}

func NewJudges(wg *sync.WaitGroup) *Judges {
	judges := &Judges{
		logger:    nil,
		mainDB:    nil,
		waitGroup: wg,
		stop:      make(chan struct{}),
		verdicts:  make(chan Verdict),
	}

	return judges
}

func (j *Judges) Start(cfg config.JudgesConfig) error {
	// set up common dependencies
	if err := j.SetupLogger(cfg); err != nil {
		return err
	}
	if err := j.ConnectMainDB(cfg); err != nil {
		return err
	}
	j.waitGroup.Add(1)

	go j.SubmissionUpdater()

	// set up judges
	j.selectionJudge = NewSelectionJudge(j.logger, j.mainDB, j.waitGroup, j.stop, j.verdicts)

	// start judges
	if err := j.selectionJudge.Start(cfg.SelectionJudgeConfig); err != nil {
		return err
	}

	return nil
}

func (j *Judges) Stop() {
	j.selectionJudge.Stop()
	//j.dataManipulationJudge.Stop()
	//j.schemaManipulationJudge.Stop()
	close(j.stop)
}

func (j *Judges) SetupLogger(cfg config.JudgesConfig) error {
	logger, err := cfg.LoggerConfig.Build()
	if err != nil {
		return err
	}

	j.logger = logger

	return nil
}

func (j *Judges) ConnectMainDB(cfg config.JudgesConfig) error {
	connectionString := cfg.MainDBConfig.ConnectionString()

	db, err := connectDB(connectionString)
	if err != nil {
		return err
	}

	j.mainDB = db

	j.logger.Info(
		"main database connected",
		zap.String("connection_string", connectionString),
	)

	return nil
}

func (j *Judges) UpdateSubmissionReviewInfo(ID, statusID int, reviewerMessage string) error {
	query := templates.UpdateSubmissionReviewInfo
	_, err := j.mainDB.Exec(query, statusID, reviewerMessage, ID)
	if err != nil {
		return err
	}
	return nil
}

func (j *Judges) SubmissionUpdater() {
	defer func() {
		j.logger.Info("stopped submission updater")
		j.waitGroup.Done()
	}()
	for {
		select {
		case <-j.stop:
			return
		case v := <-j.verdicts:
			if err := j.UpdateSubmissionReviewInfo(
				v.SubmissionID,
				v.SubmissionStatusID,
				v.ReviewerMessage,
			); err != nil {
				j.logger.Error(
					"submission update failed",
					zap.Int("submission_id", v.SubmissionID),
				)
			}
		}
	}
}
