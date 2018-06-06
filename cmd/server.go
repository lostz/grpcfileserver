package cmd

import (
	"log"

	"github.com/lostz/grpcfileserver/core"
	"github.com/spf13/cobra"
)

var path string
var port int

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run as server",
	Long: `
	grpcfile --server --port 1213 --path test

	`,
	Run: server,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVar(&port, "port", 1213, "server port")
	serverCmd.Flags().StringVar(&path, "path", "./", "path")

}

func server(cmd *cobra.Command, args []string) {
	s := core.NewServerGRPC(path, port)
	if err := s.Listen(); err != nil {
		log.Fatal(err.Error())
	}
	defer s.Close()
	return

}
