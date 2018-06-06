package core

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strconv"

	"github.com/cheggaaa/pb"
	"github.com/lostz/grpcfileserver/protocol"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// ServerGRPC ...
type ServerGRPC struct {
	server *grpc.Server
	path   string
	port   int
	pool   *pb.Pool
}

//NewServerGRPC ...
func NewServerGRPC(path string, port int) *ServerGRPC {
	s := &ServerGRPC{}
	s.port = port
	s.path = path
	s.pool, _ = pb.StartPool()
	return s
}

//Listen ...
func (s *ServerGRPC) Listen() (err error) {
	var (
		listener net.Listener
		grpcOpts = []grpc.ServerOption{}
	)
	listener, err = net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(s.port)))
	if err != nil {
		return err
	}
	s.server = grpc.NewServer(grpcOpts...)
	protocol.RegisterGRPCFileServiceServer(s.server, s)
	err = s.server.Serve(listener)
	return err
}

// Close ...
func (s *ServerGRPC) Close() {
	if s.server != nil {
		s.server.Stop()
	}

}

// Download ...
func (s *ServerGRPC) Download(f *protocol.File, stream protocol.GRPCFileService_DownloadServer) (err error) {
	var (
		writing = true
		buf     []byte
		n       int
		file    *os.File
	)

	file, err = os.Open(f.Filepath)
	if err != nil {
		if os.IsNotExist(err) {
			err = stream.Send(&protocol.Chunk{
				Filepath:       f.Filepath,
				Content:        []byte{},
				SizeInBytes:    int64(0),
				SizeTotalBytes: int64(0),
				IsLastChunk:    true,
			})
		}
		return

	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return
	}
	size := stat.Size()
	buf = make([]byte, 1<<12)
	bar := pb.New64(size).Postfix(" " + f.Filepath)
	bar.Units = pb.U_BYTES
	s.pool.Add(bar)
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
			Filepath:       f.Filepath,
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
	log.Println("finish")
	bar.Finish()
	return

}

//Upload ...
func (s *ServerGRPC) Upload(stream protocol.GRPCFileService_UploadServer) error {
	var filepath string
	firstChunk := true
	var fd *os.File
	var err error
	for {
		nk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				goto END
			}
			err = errors.Wrapf(err,
				"failed unexpectadely while reading chunks from stream")
			return err
		}
		filepath = nk.Filepath
		if nk.SizeInBytes != int64(len(nk.Content)) {
			return fmt.Errorf("%v == nk.SizeInBytes != int64(len(nk.Data)) == %v", nk.SizeInBytes, int64(len(nk.Content)))
		}
		if firstChunk {
			if filepath != "" {
				fd, err = os.Create(path.Join(s.path, filepath))
				if err != nil {
					return err
				}
				defer fd.Close()
			}
			firstChunk = false
		}
		err = writeToFd(fd, nk.Content)
		if err != nil {
			return err
		}
		if nk.IsLastChunk {
			goto END
		}

	}
END:
	err = stream.SendAndClose(&protocol.UploadStatus{
		Code:     protocol.StatusCode_Ok,
		Filepath: filepath,
	})

	if err != nil {
		err = errors.Wrapf(err,
			"failed to send status code")
		return err
	}
	return err

}

func writeToFd(fd *os.File, data []byte) error {
	w := 0
	n := len(data)
	for {
		nw, err := fd.Write(data[w:])
		if err != nil {
			return err
		}
		w += nw
		if nw >= n {
			return nil
		}
	}
}
