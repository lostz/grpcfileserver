package cmd

import (
	"context"
	"log"

	"github.com/lostz/grpcfileserver/core"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "upload file",
	Long: `
	grpcile --upload --host 0.0.0.0:1213 --chunkSize 4096 --file test.go
	`,
	Run: upload,
}

var (
	address   string
	file      string
	chunkSize int
)

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringVar(&address, "host", "0.0.0.0:1213", "remote server")
	uploadCmd.Flags().StringVar(&file, "file", "", "upload file")
	uploadCmd.Flags().IntVar(&chunkSize, "chunkSize", 1<<12, "chunk size ")
}

func upload(cmd *cobra.Command, args []string) {
	c := core.ClientGRPCConfig{
		Address:   address,
		ChunkSize: chunkSize,
	}

	client, err := core.NewClientGRPC(c)
	if err != nil {
		log.Fatal(err.Error())

	}
	_, err = client.UploadFile(context.Background(), file)
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer client.Close()
	return

}
