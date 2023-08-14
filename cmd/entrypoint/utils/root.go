package utils

import (
	"github.com/spf13/cobra"
)

var entryFlags *EntryFlags

func init() {
	entryFlags = &EntryFlags{}
	rootCmd.Flags().StringVar(&entryFlags.waitFile, "wait", "", "entrypoint --wait /var/run/1")
	//	rootCmd.Flags().StringVar(&entryFlags.waitFileContent, "waitFileContent", "", "entrypoint --waitFileContent")
	rootCmd.Flags().StringVar(&entryFlags.out, "out", "", "entrypoint --out /var/run/out")
	rootCmd.Flags().StringVar(&entryFlags.command, "command", "", "entrypoint --command bash")
	//	rootCmd.Flags().StringVar(&entryFlags.quitContent, "quit", "-1", "entrypoint --quit -2")
	//	rootCmd.Flags().StringVar(&entryFlags.encodeFile, "encodefile", "-1", "entrypoint --encodefile /var/run/1")
}

var rootCmd = &cobra.Command{
	Use:   "entrypoint",
	Short: "Generic entrypoint program",
	Long:  "An entry for tasks to be executed in a unified order",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return entryFlags.validate()
	},
	Run: func(cmd *cobra.Command, args []string) {
		watchWaitFile()
		execCmdAndArgs(args)
	},
}

func main() {
	//rootCmd.AddCommand()
}

//
//func InitCmd() {
//	var flags = &EntryFlags{}
//	if err := rootCmd.Execute(); err != nil {
//		log.Fatal(err)
//	}
//}
