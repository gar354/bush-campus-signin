package serveQr

import (
	"github.com/gar354/bush-campus-signin/broadcast"

	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type Server struct {
	url        string
	uuid       string
	imgData    []byte
	Broadcast  broadcast.Broadcaster
	Upgrader   websocket.Upgrader
	mu         sync.Mutex
	qrPassword string
}

func New(qrPassword string) *Server {
	s := Server{
		Broadcast: broadcast.New(),
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		qrPassword: qrPassword,
	}
	return &s
}

func (s *Server) RefreshQr() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.uuid = uuid.NewString()
	s.url = fmt.Sprintf("%s?UUID=%s", os.Getenv("URL"), s.uuid)
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

func (s *Server) CheckUUID(uuid string) bool {
	return s.uuid == uuid
}

func (s *Server) GetIMGData() []byte {
	return s.imgData
}

func (s *Server) CheckPassword(password string) bool {
	return s.qrPassword == password
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
