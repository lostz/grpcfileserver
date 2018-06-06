package cmd

import "github.com/spf13/cobra"
import "log"

var rootCmd = &cobra.Command{
	Use:     "grpcfile",
	Short:   " upload or download file",
	Long:    ` upload or download file`,
	Version: "0.1.0",
}

// Execute ..
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}

}

func init() {
	cobra.OnInitialize()

}
