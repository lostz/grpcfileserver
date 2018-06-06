package cmd

import (
	"context"
	"log"

	"github.com/lostz/grpcfileserver/core"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download file",
	Long: `
	    grpcile --download --host 0.0.0.0:1213 --chunkSize 4096 --file /storage/test.go --write /data/test2/test.go
		    `,
	Run: download,
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().StringVar(&address, "host", "0.0.0.0:1213", "remote server")
	downloadCmd.Flags().StringVar(&file, "file", "", "download file")
	downloadCmd.Flags().StringVar(&path, "write", "", "write path")
	downloadCmd.Flags().IntVar(&chunkSize, "chunkSize", 1<<12, "chunk size ")
}

func download(cmd *cobra.Command, args []string) {
	c := core.ClientGRPCConfig{
		Address:   address,
		ChunkSize: chunkSize,
	}

	client, err := core.NewClientGRPC(c)
	if err != nil {
		log.Fatal(err.Error())

	}
	err = client.DownloadFile(context.Background(), file, path)
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer client.Close()
	return

}
