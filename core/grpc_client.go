package core

import (
	"context"
	"io"
	"os"
	"path"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/lostz/grpcfileserver/protocol"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Stats ...
type Stats struct {
	StartedAt  time.Time
	FinishedAt time.Time
}

//ClientGRPC ...
type ClientGRPC struct {
	conn      *grpc.ClientConn
	client    protocol.GRPCFileServiceClient
	chunkSize int
	pool      *pb.Pool
}

//ClientGRPCConfig ...
type ClientGRPCConfig struct {
	Address   string
	ChunkSize int
	Compress  bool
}

//NewClientGRPC ...
func NewClientGRPC(cfg ClientGRPCConfig) (c *ClientGRPC, err error) {
	c = &ClientGRPC{}
	grpcOpts := []grpc.DialOption{}
	if cfg.Address == "" {
		err = errors.Errorf("address must be specified")
		return
	}
	c.pool, _ = pb.StartPool()
	if cfg.Compress {
		grpcOpts = append(grpcOpts,
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	}
	grpcOpts = append(grpcOpts, grpc.WithInsecure())
	switch {
	case cfg.ChunkSize == 0:
		err = errors.Errorf("ChunkSize must be specified")
		return
	case cfg.ChunkSize > (1 << 22):
		err = errors.Errorf("ChunkSize must be < than 4MB")
		return
	default:
		c.chunkSize = cfg.ChunkSize
	}
	c.conn, err = grpc.Dial(cfg.Address, grpcOpts...)
	if err != nil {
		return c, err
	}
	c.client = protocol.NewGRPCFileServiceClient(c.conn)
	return

}

// DownloadFile ...
func (c *ClientGRPC) DownloadFile(ctx context.Context, target, filepath string) (err error) {
	var size int64
	var writing = true
	var bar *pb.ProgressBar
	file, err := os.Create(filepath)
	if err != nil {
		return
	}
	defer file.Close()
	request := &protocol.File{
		Filepath: target,
	}
	stream, err := c.client.Download(ctx, request)
	if err != nil {
		return
	}
	for writing {
		res, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				writing = false
				err = nil
				continue
			}
			err = errors.Wrapf(err, "errored while copying from file to buf")
			return err
		}
		if res.IsLastChunk == true && res.SizeTotalBytes == 0 {
			return os.ErrNotExist
		}
		if size == 0 {
			bar = pb.New64(res.SizeTotalBytes).Postfix(" " + res.Filepath)
			bar.Units = pb.U_BYTES
			c.pool.Add(bar)
			size = res.SizeTotalBytes
		}
		bar.Add64(res.SizeInBytes)
		err = writeToFd(file, res.Content)
		if err != nil {
			return err
		}
		if res.IsLastChunk {
			bar.Finish()
			return nil
		}

	}

	return
}

// UploadFile ...
func (c *ClientGRPC) UploadFile(ctx context.Context, f string) (stats Stats, err error) {
	var (
		writing = true
		buf     []byte
		n       int
		file    *os.File
	)

	file, err = os.Open(f)
	if err != nil {
		return
	}
	defer file.Close()
	stream, err := c.client.Upload(ctx)
	if err != nil {
		return
	}
	stat, err := file.Stat()
	if err != nil {
		return

	}
	size := stat.Size()
	bar := pb.New64(size).Postfix(" " + f)
	bar.Units = pb.U_BYTES
	c.pool.Add(bar)
	defer stream.CloseSend()
	stats.StartedAt = time.Now()
	buf = make([]byte, c.chunkSize)
	pathBase := path.Base(f)
	for writing {
		n, err = file.Read(buf)
		if err != nil {
			if err == io.EOF {
				writing = false
				err = nil
				continue
			}

			err = errors.Wrapf(err,
				"errored while copying from file to buf")
			return
		}
		bar.Add64(int64(n))
		err = stream.Send(&protocol.Chunk{
			Filepath:       pathBase,
			Content:        buf[:n],
			SizeTotalBytes: size,
			SizeInBytes:    int64(n),
			IsLastChunk:    writing == false,
		})
		if err != nil {
			err = errors.Wrapf(err,
				"failed to send chunk via stream")
			return
		}
	}
	stats.FinishedAt = time.Now()
	status, err := stream.CloseAndRecv()
	if err != nil {
		return
	}
	if status.Code != protocol.StatusCode_Ok {
		err = errors.New("upload faild")

	}
	bar.Finish()
	return

}

// Close ...
func (c *ClientGRPC) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.pool.RefreshRate = 500 * time.Millisecond
	c.pool.Stop()
}
