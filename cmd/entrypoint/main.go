package main

import (
	"github.com/daicheng123/ordertask-operator/cmd/entrypoint/utils"
	"os"
)

func main() {
	if err := utils.RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
