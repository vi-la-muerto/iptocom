package device

import (
	"io"
	extypes "vs/iptocom/types"

	log "github.com/sirupsen/logrus"
	serial "github.com/tarm/goserial"
)

type SerialPort struct {
	serial.Config
	device io.ReadWriteCloser
	buffer []byte
}

func (s *SerialPort) CreateConfig(Name string, Baud int) {
	s.Config = serial.Config{Name: Name, Baud: Baud}
}

func (s *SerialPort) OpenPort() error {

	var err error = nil

	s.buffer = make([]byte, 1024)
	s.device, err = serial.OpenPort(&s.Config)

	return err
}

func (s *SerialPort) ClosePort() error {
	if s.device != nil {
		return s.device.Close()
	} else {
		return nil
	}
}

func (s *SerialPort) ReadFromDevice(result chan extypes.ReadingResult, goOn chan bool) {

	log.WithFields(log.Fields{
		"package":    "device",
		"func":       "ReadFromDevice",
		"serialPort": s.Config,
	}).Debug("Try to read from serial port")

	for <-goOn {

		quantity, err := s.device.Read(s.buffer)

		log.WithFields(log.Fields{
			"package":  "device",
			"func":     "ReadFromDevice",
			"quantity": quantity,
			"error":    err,
			"buffer":   s.buffer[:quantity],
		}).Debug("Reading serial port was end")

		result <- extypes.ReadingResult{QuantityBytes: quantity, Error: err}
	}
}

func (s *SerialPort) GetBuffer() []byte {
	return s.buffer
}

func (s *SerialPort) WriteToDevice(buffer []byte, readingResult extypes.ReadingResult) (error, error) {

	log.WithFields(log.Fields{
		"package":    "device",
		"func":       "WriteToDevice",
		"serialPort": s.Config,
	}).Debug("Try to write from serial port")

	var (
		readingError error = nil
		writingError error = nil
	)

	if readingResult.Error != nil {
		readingError = readingResult.Error
		if readingResult.Error != io.EOF {
			log.WithFields(log.Fields{
				"package":       "device",
				"func":          "WriteToDevice",
				"readingResult": readingResult,
			}).Error("Reading result has error")
		}
	} else {
		_, writingError = s.device.Write(buffer[0:readingResult.QuantityBytes])

		log.WithFields(log.Fields{
			"package":       "device",
			"func":          "WriteToDevice",
			"error":         writingError,
			"readingResult": readingResult,
		}).Debug("Write to serial port was success")
	}

	return readingError, writingError
}
