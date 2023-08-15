package utils

import (
	"errors"
	"golang.org/x/sys/execabs"
	"os"
	"path/filepath"
	"time"
)

func watchWaitFile() error {
	ticker := time.NewTicker(entryFlags.scanInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		f, err := os.Stat(entryFlags.waitFile)
		if err == nil {
			if f.IsDir() {
				return errors.New("wait file cloud not be directory!")
			}
			return nil
		} else if errors.Is(err, os.ErrNotExist) {
			continue
		} else {
			return err
		}
	}
}

func execCmdAndArgs(args []string) error {
	var logFile *os.File
	if entryFlags.out == "" {
		logFile = os.Stdout
	} else {
		outfilePath := filepath.Join(getWorkDir(), entryFlags.out)
		lf, err := os.OpenFile(outfilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		logFile = lf
		defer logFile.Close()
	}
	exec := execabs.Command(entryFlags.command, args...)
	exec.Stdout = logFile
	exec.Stderr = logFile
	return exec.Run()
}

func getWorkDir() string {
	executablePath := os.Args[0]
	return filepath.Dir(executablePath)
}
