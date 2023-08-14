package utils

import (
	"errors"
	"os"
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
		//return
		lf, err := os.OpenFile(entryFlags.out, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		logFile = lf
		defer logFile.Close()
	}

}
