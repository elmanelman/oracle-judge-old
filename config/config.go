package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-ozzo/ozzo-validation/v3"
	"github.com/go-ozzo/ozzo-validation/v3/is"
	"go.uber.org/zap"
	"io/ioutil"
	"path/filepath"
)

type OracleConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	SID      string `json:"sid"`
}

func (c *OracleConfig) ConnectionString() string {
	return fmt.Sprintf("%s/%s@%s:%s/%s", c.Username, c.Password, c.Host, c.Port, c.SID)
}

func (c *OracleConfig) Validate() error {
	return validation.ValidateStruct(
		c,
		validation.Field(&c.Username, validation.Required),
		validation.Field(&c.Password, validation.Required),
		validation.Field(&c.Host, validation.Required, is.Host),
		validation.Field(&c.Port, validation.Required, is.Port),
		validation.Field(&c.SID, validation.Required),
	)
}

type SelectionDBConfig struct {
	Name   string       `json:"name"`
	Config OracleConfig `json:"config"`
}

func (c *SelectionDBConfig) Validate() error {
	if err := c.Config.Validate(); err != nil {
		return err
	}
	if c.Config.Username != "JUDGE_"+c.Name {
		return errors.New("invalid selection DB username")
	}
	return nil
}

const (
	minFetchPeriod   = 100
	minReviewerCount = 0
)

type SelectionJudgeConfig struct {
	DBConfigs []SelectionDBConfig `json:"dbs"`

	FetchPeriod   int `json:"fetch_period"`
	ReviewerCount int `json:"reviewer_count"`
}

func (c *SelectionJudgeConfig) Validate() error {
	for _, sc := range c.DBConfigs {
		if err := sc.Validate(); err != nil {
			return err
		}
	}
	return validation.ValidateStruct(
		c,
		validation.Field(&c.FetchPeriod, validation.Required, validation.Min(minFetchPeriod)),
		validation.Field(&c.ReviewerCount, validation.Min(minReviewerCount)),
	)
}

type ManipulationJudgeConfig struct {
	FetchPeriod   int `json:"fetch_period"`
	ReviewerCount int `json:"reviewer_count"`
}

func (c *ManipulationJudgeConfig) Validate() error {
	return validation.ValidateStruct(
		c,
		validation.Field(&c.FetchPeriod, validation.Required, validation.Min(minFetchPeriod)),
		validation.Field(&c.ReviewerCount, validation.Min(minReviewerCount)),
	)
}

type JudgesConfig struct {
	LoggerConfig zap.Config `json:"logger"`

	MainDBConfig OracleConfig `json:"main_db"`

	SelectionJudgeConfig          SelectionJudgeConfig    `json:"selection_judge"`
	DataManipulationJudgeConfig   ManipulationJudgeConfig `json:"data_manipulation_judge"`
	SchemaManipulationJudgeConfig ManipulationJudgeConfig `json:"schema_manipulation_judge"`
}

func (c *JudgesConfig) Validate() error {
	if err := c.MainDBConfig.Validate(); err != nil {
		return err
	}
	if err := c.SelectionJudgeConfig.Validate(); err != nil {
		return err
	}
	if err := c.DataManipulationJudgeConfig.Validate(); err != nil {
		return err
	}
	if err := c.SchemaManipulationJudgeConfig.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *JudgesConfig) LoadFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	switch ext := filepath.Ext(defaultConfigFile); ext {
	case ".json":
		if err := c.loadFromJSON(data); err != nil {
			return err
		}
		return c.Validate()
	default:
		return fmt.Errorf("unknown configuration file extension: %s", ext)
	}
}

func (c *JudgesConfig) loadFromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

const defaultConfigFile = "config.json"

func (c *JudgesConfig) LoadDefault() error {
	return c.LoadFromFile(defaultConfigFile)
}
