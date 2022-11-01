package source

import (
	"reflect"
	"time"

	"github.com/grepplabs/mqtt-proxy/pkg/log"
)

func Watch(logger log.Logger, ch chan ServerCerts, refresh time.Duration, init *ServerCerts, loadFn func() (*ServerCerts, error), changedFn func()) {
	once := refresh <= 0

	if refresh < time.Second {
		refresh = time.Second
	}
	logger.Infof("server cert watch is started, refresh interval %s", refresh)

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
				if once && init != nil {
					// init value is set, so assume it was already sent to channel
					logger.Info("server cert watch is disabled")
					return
				}
				time.Sleep(refresh)
				continue
			}
		}

		ch <- *next
		last = next

		if changedFn != nil {
			changedFn()
		}
		if once {
			logger.Info("server cert watch is disabled")
			return
		}
		time.Sleep(refresh)
	}
}
