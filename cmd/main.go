package main

import (
	"os"
	"os/signal"
	"syscall"

	config "github.com/jonkeyguan/vfio-device-plugin/pkg/config"
	device_manager "github.com/jonkeyguan/vfio-device-plugin/pkg/device-manager"
	log "github.com/jonkeyguan/vfio-device-plugin/pkg/log"
)

const (
	DeviceAccessPermissions = "rwm"
)

func main() {
	stop := make(chan struct{})
	done := make(chan struct{})

	logger := log.DefaultLogger()

	resourceConfig, err := config.NewResourceConfig()
	if err != nil {
		logger.Reason(err).Error("Failed to create resource config")
		return
	}

	deviceController := device_manager.NewDeviceController(DeviceAccessPermissions, resourceConfig)

	go deviceController.Run(stop, done)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	logger.Info("Received shutdown signal")

	close(stop)

	<-done

	logger.Info("Device Controller exited, program ending")
}
