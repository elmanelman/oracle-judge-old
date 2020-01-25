package main

import (
	"fmt"
	"github.com/elmanelman/oracle-judge/config"
	"github.com/elmanelman/oracle-judge/judge"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	cfg := config.JudgesConfig{}
	if err := cfg.LoadDefault(); err != nil {
		log.Fatal(err)
	}

	wg := new(sync.WaitGroup)

	j := judge.NewJudges(wg)
	setupSigtermHandler(j)
	if err := j.Start(cfg); err != nil {
		log.Fatal(err)
	}

	wg.Wait()
}

func setupSigtermHandler(judges *judge.Judges) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Print("\n")
		judges.Stop()
	}()
}
