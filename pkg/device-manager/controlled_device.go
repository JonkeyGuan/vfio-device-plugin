package device_manager

import (
	"math"
	"time"

	"github.com/jonkeyguan/vfio-device-plugin/pkg/log"
)

var defaultBackoffTime = []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second}

type controlledDevice struct {
	devicePlugin Device
	started      bool
	stopChan     chan struct{}
	backoff      []time.Duration
}

func (c *controlledDevice) Start() {
	if c.started {
		return
	}

	stop := make(chan struct{})

	logger := log.DefaultLogger()
	dev := c.devicePlugin
	deviceName := dev.GetDeviceName()
	logger.Infof("Starting a device plugin for device: %s", deviceName)
	retries := 0

	backoff := c.backoff
	if backoff == nil {
		backoff = defaultBackoffTime
	}

	go func() {
		for {
			err := dev.Start(stop)
			if err != nil {
				logger.Reason(err).Errorf("Error starting %s device plugin", deviceName)
				retries = int(math.Min(float64(retries+1), float64(len(backoff)-1)))
			} else {
				retries = 0
			}

			select {
			case <-stop:
				// Ok we don't want to re-register
				return
			case <-time.After(backoff[retries]):
				// Wait a little and re-register
				continue
			}
		}
	}()

	c.stopChan = stop
	c.started = true
}

func (c *controlledDevice) Stop() {
	if !c.started {
		return
	}
	close(c.stopChan)

	c.stopChan = nil
	c.started = false
}

func (c *controlledDevice) GetName() string {
	return c.devicePlugin.GetDeviceName()
}
