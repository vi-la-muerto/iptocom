package app

import (
	"io"
	"vs/iptocom/device"
	"vs/iptocom/tcp"
	extypes "vs/iptocom/types"

	log "github.com/sirupsen/logrus"
)

type ConfigApp struct {
	IPv4                      string
	TCPPort                   int
	SerialPort                string
	BaudSpeed                 int
	ReadTimeoutTCPConnection  int
	WriteTimeoutTCPConnection int
}

type ProxyServer struct {
	device.SerialPort
	mainServer       tcp.Server
	controlServer    tcp.Server
	useControlServer bool
}

func NewProxyServer(config *ConfigApp) ProxyServer {

	app := ProxyServer{}
	app.SerialPort.CreateConfig(config.SerialPort, config.BaudSpeed)
	app.mainServer = tcp.Server{
		Host:         config.IPv4,
		Port:         config.TCPPort,
		ReadTimeout:  config.ReadTimeoutTCPConnection,
		WriteTimeout: config.WriteTimeoutTCPConnection,
	}

	return app
}

func (s *ProxyServer) Start() (bool, error) {
	serErr := s.OpenPort()

	if serErr != nil {

		log.Error("Don't manage to open serial port")

		return false, serErr
	}

	defer s.ClosePort()

	netErr := s.mainServer.StartServer()

	if netErr != nil {

		log.Error("Don't manage to create socket listener")

		return false, netErr
	}

	defer s.mainServer.CloseConnections()

	return s.startHandling()
}

func (s *ProxyServer) Stop() {

	s.ClosePort()

	s.mainServer.CloseConnections()

	if s.useControlServer {
		s.controlServer.CloseConnections()
	}
}

func (s *ProxyServer) startHandling() (bool, error) {

	readingResultSerialPort, readingResultConnection := make(chan extypes.ReadingResult), make(chan extypes.ReadingResult)

	acceptedConnection := make(chan extypes.AcceptedConnection)

	goOnToReadSerialPort, goOnToReadConnection, goOnToAcceptConnection := make(chan bool), make(chan bool), make(chan bool)
	takeNextConnection, takedConnection := make(chan bool), make(chan bool)

	go s.ReadFromDevice(readingResultSerialPort, goOnToReadSerialPort)

	goOnToReadSerialPort <- true

	go s.mainServer.AcceptConnection(acceptedConnection, goOnToAcceptConnection)

	goOnToAcceptConnection <- true

	go s.mainServer.TakeToWorkNextConnection(takedConnection, takeNextConnection)

	var (
		deviceErr error = nil
		socketErr error = nil
		exit      bool  = false
	)

	for !exit {

		select {
		case acceptingResult := <-acceptedConnection:

			s.mainServer.AddConnectionToQueue(acceptingResult)

			goOnToAcceptConnection <- true
			takeNextConnection <- true

		case connectionInWork := <-takedConnection:

			if connectionInWork {
				go s.mainServer.ReadFromActiveConnection(readingResultConnection, goOnToReadConnection)
				goOnToReadConnection <- true
			}

		case readingResult := <-readingResultConnection:

			socketErr, deviceErr = s.WriteToDevice(s.mainServer.GetBuffer(), readingResult)
			goOnToReadConnection <- deviceErr == nil && socketErr == nil

		case readingResult := <-readingResultSerialPort:

			deviceErr, socketErr = s.mainServer.WriteToActiveConnection(s.SerialPort.GetBuffer(), readingResult)
			goOnToReadSerialPort <- deviceErr == nil && s.mainServer.ActiveConnection != nil
		}

		if deviceErr != nil {
			log.WithFields(log.Fields{
				"package": "app",
				"func":    "startHandling",
				"error":   deviceErr,
			}).Error("Error serial port.")

			return exit, deviceErr
		}

		if s.mainServer.ActiveConnection != nil && socketErr != nil {
			if socketErr == io.EOF {
				log.WithFields(log.Fields{
					"package":    "app",
					"func":       "startHandling",
					"connection": s.mainServer.ActiveConnection.RemoteAddr(),
				}).Info("Connection handling was ended.")
			} else {
				log.WithFields(log.Fields{
					"package":    "app",
					"func":       "startHandling",
					"connection": s.mainServer.ActiveConnection.RemoteAddr(),
					"error":      socketErr,
				}).Error("Unexpected socket error.")

			}

			s.mainServer.CloseActiveConnection()
			socketErr = nil
			takeNextConnection <- true
		}
	}

	return exit, nil
}
