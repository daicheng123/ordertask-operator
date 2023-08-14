package utils

import (
	"errors"
	"time"
)

const (
	defaultScanInterval = 20
)

type EntryFlags struct {
	waitFile        string
	waitFileContent string
	out             string
	command         string
	quitContent     string
	encodeFile      string
	scanInterval    time.Duration
}

func (ef *EntryFlags) validate() error {
	if len(ef.waitFile) == 0 {
		return errors.New("wait file can't be empty!")
	}

	if len(ef.out) == 0 {
		return errors.New("out file can't be empty!")
	}

	if len(ef.command) == 0 {
		return errors.New("command  can't be empty!")
	}

	if ef.scanInterval == 0 {
		ef.scanInterval = defaultScanInterval * time.Millisecond
	}
	return nil
}
