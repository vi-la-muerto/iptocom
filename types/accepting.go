package extratypes

import "net"

type AcceptedConnection struct {
	Connection net.Conn
	Error      error
}
