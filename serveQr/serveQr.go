package serveQr

import (
	"gareth/attendence/broadcast"

	"bytes"
	"fmt"
	"os"
	"io"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type Server struct {
	url          string
	uuid         string
	imgData      []byte
	Broadcast    broadcast.Broadcaster
	Upgrader     websocket.Upgrader
	mu           sync.Mutex
}

func New() Server {
	s := Server{
		Broadcast: broadcast.New(),
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
	err := s.RefreshQr()
	if err != nil {
		log.Println("Error Creating QrServer: %v", err)
	}
	return s
}

func (s *Server) RefreshQr() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.uuid = uuid.NewString()
	s.url = fmt.Sprintf("%s/form?UUID=%s", os.Getenv("URL"), s.uuid)
	log.Println(s.url)

	qrc, err := qrcode.NewWith(s.url,
		qrcode.WithEncodingMode(qrcode.EncModeByte),
		qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionQuart),
	)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	wr := nopCloser{Writer: buf}
	w2 := standard.NewWithWriter(wr, standard.WithQRWidth(10))
	if err = qrc.Save(w2); err != nil {
		panic(err)
	}

	s.imgData = buf.Bytes()

	go s.Broadcast.Send(s.imgData)

	return nil
}

func (s *Server) GetUUID() string {
	return s.uuid
}

func (s *Server) GetIMGData() []byte {
	return s.imgData
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
