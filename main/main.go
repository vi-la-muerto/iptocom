package main

import (
	"os"
	"runtime"
	"syscall"
	"time"
	"vs/iptocom/app"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	portName     string
	portBaud     int
	addressHost  string
	tcpPort      int
	debug        bool
	logFile      string
	readtimeout  int
	writetimeout int
	showHelp     bool
)

func main() {

	if runtime.GOOS != "windows" {
		log.WithFields(log.Fields{
			"package": "main",
			"func":    "main",
			"OS":      runtime.GOOS,
		}).Fatal("Application doesn't supported this operation system.")
	}

	setFlagsForParser()
	pflag.Parse()

	if showHelp {
		pflag.Usage()
		return
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		log.WithFields(log.Fields{
			"package": "main",
			"func":    "main",
			"file":    logFile,
		}).Fatal("Don'n manage open log file for appending")
	}

	log.SetOutput(f)

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	
	configApp := app.ConfigApp{
		IPv4:                      addressHost,
		TCPPort:                   tcpPort,
		SerialPort:                portName,
		BaudSpeed:                 portBaud,
		ReadTimeoutTCPConnection:  readtimeout,
		WriteTimeoutTCPConnection: writetimeout,
	}
	log.WithFields(
		log.Fields{
			"package": "main",
			"func":    "main",
			"config":  configApp,
		}).Info("Start service")

	startAppAsServer(&configApp)
}

func setFlagsForParser() {
	pflag.BoolVarP(&showHelp, "help", "",
		false,
		"Print help message")
	pflag.IntVarP(&portBaud, "baudspeed", "b",
		115200,
		"Serail port's Baud speed. Possible values: 2400|4800|9600|19200|38400|57600|115200|230400|460800|921600")
	pflag.BoolVarP(&debug, "debug", "d",
		false,
		"Enable verbose logs")
	pflag.StringVarP(&addressHost, "host", "h",
		"127.0.0.1",
		"IP address TCP server.")
	pflag.StringVarP(&logFile, "logpath", "l",
		"app.log",
		"File for logs. If don't denife message would write to STDOUT")
	pflag.IntVarP(&tcpPort, "port", "p",
		7070,
		"TCP port which will be open on host")
	pflag.IntVarP(&readtimeout, "readtime", "r",
		1500,
		"Read	 timeout for TCP connection")
	pflag.StringVarP(&portName, "serilaport", "s",
		"COM3",
		"Serial port name. Example:COM1")
	pflag.IntVarP(&writetimeout, "writetime", "w",
		1500,
		"Write timeout for TCP connection")
}

func startAppAsServer(configApp *app.ConfigApp) {

	var (
		err  error = nil
		exit       = false
	)

	proxy := app.NewProxyServer(configApp)

	for !exit {

		log.WithFields(log.Fields{
			"package": "main",
			"func":    "startAppAsServer",
		}).Info("Start the service")

		exit, err = proxy.Start()
		if err != nil {
			
			var errDescription string

			switch err{
			case syscall.ERROR_OPERATION_ABORTED:
				errDescription = "COM port was aborted"
			case syscall.ERROR_FILE_NOT_FOUND:
				errDescription = "Not found COM-port. Check cashbox. It can be disabled or port name was defined not correctly"
			default:
				errDescription = "Unexpected error"
			}
		
			log.WithFields(log.Fields{
				"package": "main",
				"func":    "startAppAsServer",
				"config":  configApp,
				"error":   err,
			}).Error(errDescription)

			log.WithFields(log.Fields{
				"package": "main",
				"func":    "startAppAsServer",
			}).Info("Stop the service")

			proxy.Stop()
			time.Sleep(5 * time.Second)
		}
	}
}
