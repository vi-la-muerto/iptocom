package tcp

import (
	"fmt"
	"net"
	"time"
	extypes "vs/iptocom/types"

	log "github.com/sirupsen/logrus"
)

type Server struct {
	Host             string
	Port             int
	listener         net.Listener
	QueueConnection  []extypes.AcceptedConnection
	ActiveConnection net.Conn
	buffer           []byte
	ReadTimeout      int
	WriteTimeout     int
}

func (s *Server) StartServer() error {

	var err error = nil

	s.buffer = make([]byte, 1024)
	s.listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port))

	return err
}

func (s *Server) CloseConnections() {

	log.WithFields(log.Fields{
		"package": "tcp",
		"func":    "CloseConnections",
	}).Debug("Try to close active connections of queue")

	if s.ActiveConnection != nil {
		s.ActiveConnection.Close()
	}

	for _, elem := range s.QueueConnection {
		elem.Connection.Close()
	}

	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) CloseActiveConnection() {

	s.ActiveConnection.Close()
	s.ActiveConnection = nil
}

func (s *Server) AcceptConnection(result chan extypes.AcceptedConnection, goOn chan bool) {

	log.WithFields(log.Fields{
		"package": "tcp",
		"func":    "AcceptConnection",
		"server":  s.listener.Addr(),
	}).Debug("Try accepting active connections")

	for <-goOn {
		connection, err := s.listener.Accept()

		log.WithFields(log.Fields{
			"package":                "tcp",
			"func":                   "AcceptConnection",
			"connection remote addr": connection.RemoteAddr(),
			"error":                  err,
		}).Info("Accept a new connection")

		result <- extypes.AcceptedConnection{Connection: connection, Error: err}
	}
}

func (s *Server) AddConnectionToQueue(acceptedConnection extypes.AcceptedConnection) {

	log.WithFields(log.Fields{
		"package": "tcp",
		"func":    "AddConnectionToQueue",
	}).Debug("Add new connection to queue")

	s.QueueConnection = append(s.QueueConnection, acceptedConnection)
}

func (s *Server) TakeToWorkNextConnection(takedConnection chan bool, goOn chan bool) {

	log.WithFields(log.Fields{
		"package": "tcp",
		"func":    "TakeToWorkNextConnection",
	}).Debug("Start taking next connection from queue")

	for <-goOn {

		if len(s.QueueConnection) > 0 && s.ActiveConnection == nil {
			s.ActiveConnection = s.QueueConnection[0].Connection
			s.QueueConnection = s.QueueConnection[1:]
			takedConnection <- true

			log.WithFields(log.Fields{
				"package":                "tcp",
				"connection remote addr": s.ActiveConnection.RemoteAddr(),
				"func":                   "TakeToWorkNextConnection",
			}).Info("Taked next connection from queue")

		} else if len(s.QueueConnection) == 0 {
			takedConnection <- false

			log.WithFields(log.Fields{
				"package": "tcp",
				"func":    "TakeToWorkNextConnection",
			}).Debug("Queue of connection is empty")

		} else {
			takedConnection <- false

			log.WithFields(log.Fields{
				"package": "tcp",
				"func":    "TakeToWorkNextConnection",
				"queue":   s.QueueConnection,
			}).Warn("Don't manage to take next connection from queue")
		}
	}
}

func (s *Server) WriteToActiveConnection(buffer []byte, readingResult extypes.ReadingResult) (error, error) {

	log.WithFields(log.Fields{
		"package": "tcp",
		"func":    "WriteToActiveConnection",
		"server":  s.listener.Addr(),
	}).Debug("Try to write to active connection")

	var (
		readingError error
		writingError error
	)

	if readingResult.Error != nil {
		readingError = readingResult.Error

		log.WithFields(log.Fields{
			"package":       "tcp",
			"func":          "WriteToActiveConnection",
			"readingResult": readingResult,
		}).Error("Reading result has error")

	} else {
		if s.WriteTimeout != 0 {
			s.ActiveConnection.SetWriteDeadline(time.Now().Add(time.Millisecond * time.Duration(s.WriteTimeout)))
		}

		_, writingError = s.ActiveConnection.Write(buffer[0:readingResult.QuantityBytes])

		log.WithFields(log.Fields{
			"package":       "tcp",
			"func":          "WriteToActiveConnection",
			"error":         writingError,
			"readingResult": readingResult,
		}).Debug("Write to active connection was success")

	}

	return readingError, writingError
}

func (s *Server) ReadFromActiveConnection(result chan extypes.ReadingResult, goOn chan bool) {

	log.WithFields(log.Fields{
		"package": "tcp",
		"func":    "ReadFromActiveConnection",
		"server":  s.listener.Addr(),
	}).Debug("Reading from active connection")

	for <-goOn {

		if s.ReadTimeout != 0 {
			s.ActiveConnection.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(s.ReadTimeout)))
		}
		quantity, err := s.ActiveConnection.Read(s.buffer)

		log.WithFields(log.Fields{
			"package":  "tcp",
			"func":     "ReadFromActiveConnection",
			"quantity": quantity,
			"error":    err,
			"buffer":   s.buffer[:quantity],
		}).Debug("Reading active connection was end")

		result <- extypes.ReadingResult{QuantityBytes: quantity, Error: err}
	}
}

func (s *Server) GetBuffer() []byte {
	return s.buffer
}
