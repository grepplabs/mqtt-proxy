package mqttserver

import (
	"fmt"
	"github.com/grepplabs/mqtt-proxy/pkg/log"
	"net"
	"sync"
)

var (
	uniqNameMu   sync.Mutex
	uniqNameNext = make(map[string]int)
)

type loggingConn struct {
	name string
	net.Conn
	logger log.Logger
}

//noinspection GoUnusedExportedFunction
func NewLoggingConnFunc(baseName string, logger log.Logger) func(c net.Conn) net.Conn {
	return func(c net.Conn) net.Conn {
		return newLoggingConn(baseName, logger, c)
	}
}

func newLoggingConn(baseName string, logger log.Logger, c net.Conn) net.Conn {
	uniqNameMu.Lock()
	defer uniqNameMu.Unlock()
	uniqNameNext[baseName]++
	return &loggingConn{
		name:   fmt.Sprintf("%s-%d", baseName, uniqNameNext[baseName]),
		Conn:   c,
		logger: logger,
	}
}

func (c *loggingConn) Write(p []byte) (n int, err error) {
	c.logger.Printf("%s.Write(%d) = ....", c.name, len(p))
	n, err = c.Conn.Write(p)
	c.logger.Printf("%s.Write(%d) = %d, %v", c.name, len(p), n, err)
	return
}

func (c *loggingConn) Read(p []byte) (n int, err error) {
	c.logger.Printf("%s.Read(%d) = ....", c.name, len(p))
	n, err = c.Conn.Read(p)
	c.logger.Printf("%s.Read(%d) = %d, %v", c.name, len(p), n, err)
	return
}

func (c *loggingConn) Close() (err error) {
	c.logger.Printf("%s.Close() = ...", c.name)
	err = c.Conn.Close()
	c.logger.Printf("%s.Close() = %v", c.name, err)
	return
}
