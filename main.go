package main

import (
	"github.com/elmanelman/oracle-judge/config"
	"log"
)

func main() {
	cfg := config.JudgeConfig{}
	if err := cfg.LoadDefault(); err != nil {
		log.Fatal("failed to load configuration: ", err)
	}
}
