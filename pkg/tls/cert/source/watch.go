package source

import (
	"reflect"
	"time"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
)

func Watch(logger log.Logger, ch chan ServerCerts, refresh time.Duration, init *ServerCerts, loadFn func() (*ServerCerts, error)) {
	once := refresh <= 0

	if refresh < time.Second {
		refresh = time.Second
	}

	var last = init
	for {
		next, err := loadFn()
		if err != nil {
			logger.WithError(err).Errorf("cannot load certificates %v", err)
			time.Sleep(refresh)
			continue
		}
		if last != nil {
			if reflect.DeepEqual(next.Checksum, last.Checksum) {
				time.Sleep(refresh)
				continue
			}
		}

		ch <- *next
		last = next

		if once {
			logger.Info("TLS server cert watch is disabled")
			return
		}
		time.Sleep(refresh)
	}
}
