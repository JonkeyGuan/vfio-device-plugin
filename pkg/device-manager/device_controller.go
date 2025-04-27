package device_manager

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	k8scli "k8s.io/client-go/kubernetes/typed/core/v1"

	config "github.com/jonkeyguan/vfio-device-plugin/pkg/config"
	log "github.com/jonkeyguan/vfio-device-plugin/pkg/log"
)

type DeviceController struct {
	startedPlugins      map[string]controlledDevice
	startedPluginsMutex sync.Mutex
	permissions         string
	backoff             []time.Duration
	resourceConfig      *config.ResourceConfig
	stop                chan struct{}
	clientset           k8scli.CoreV1Interface
}

func NewDeviceController(
	permissions string,
	resourceConfig *config.ResourceConfig,
) *DeviceController {

	controller := &DeviceController{
		startedPlugins: map[string]controlledDevice{},
		permissions:    permissions,
		backoff:        defaultBackoffTime,
		resourceConfig: resourceConfig,
	}

	return controller
}

func (c *DeviceController) Run(stop chan struct{}, done chan<- struct{}) error {
	logger := log.DefaultLogger()

	defer close(done)

	discoverConfiguredVfioDevices := c.discoverConfiguredVfioDevices()
	devicePlugins := c.buildDevicePlugins(discoverConfiguredVfioDevices)

	// start all device plugins
	func() {
		c.startedPluginsMutex.Lock()
		defer c.startedPluginsMutex.Unlock()
		for _, dev := range devicePlugins {
			logger.Infof("Starting device plugin for %s", dev.GetDeviceName())
			c.startDevice(dev.GetDeviceName(), dev)
		}
	}()
	logger.Info("Starting device plugin controller")

	// keep running until stop
	<-stop

	// stop all device plugins
	func() {
		c.startedPluginsMutex.Lock()
		defer c.startedPluginsMutex.Unlock()
		for name := range c.startedPlugins {
			c.stopDevice(name)
		}
	}()
	logger.Info("Shutting down device plugin controller")
	return nil
}

func (c *DeviceController) buildDevicePlugins(pciDeviceMap map[string][]*PCIDevice) []Device {
	var devices []Device
	for pciResourceName, pciDevices := range pciDeviceMap {
		log.DefaultLogger().Infof("Discovered PCIs %d devices on the node for the resource: %s", len(pciDevices), pciResourceName)
		devices = append(devices, NewPCIDevicePlugin(pciDevices, pciResourceName))
	}
	return devices
}

// discoverConfiguredVfioDevices returns a map of resourceName to a slice of PCIDevice
func (c *DeviceController) discoverConfiguredVfioDevices() map[string][]*PCIDevice {
	initHandler()

	logger := log.DefaultLogger()
	configuredDeviceMap := c.buildConfiguredDeviceMap()

	pciDeviceMap := make(map[string][]*PCIDevice)
	err := filepath.Walk(pciBasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Reason(err).Errorf("failed to walk path: %s", path)
		}

		if info.IsDir() {
			return nil
		}

		pciAddress := info.Name()

		pciID, err := Handler.GetDevicePCIID(pciBasePath, pciAddress)
		if err != nil {
			logger.Reason(err).Errorf("failed get vendor:device ID for device: %s", pciAddress)
			return nil
		}

		if resourceName, supported := configuredDeviceMap[pciAddress]; supported {
			// check device driver
			driver, err := Handler.GetDeviceDriver(pciBasePath, pciAddress)
			if err != nil || driver != "vfio-pci" {
				return nil
			}

			pcidev := &PCIDevice{
				pciID:      pciID,
				pciAddress: pciAddress,
			}
			iommuGroup, err := Handler.GetDeviceIOMMUGroup(pciBasePath, pciAddress)
			if err != nil {
				return nil
			}
			pcidev.iommuGroup = iommuGroup
			pcidev.driver = driver
			pcidev.numaNode = Handler.GetDeviceNumaNode(pciBasePath, pciAddress)
			pciDeviceMap[resourceName] = append(pciDeviceMap[resourceName], pcidev)
			logger.Infof("Discovered device %s with resource name %s", pciAddress, resourceName)
		}
		return nil
	})
	if err != nil {
		logger.Reason(err).Errorf("failed to discover vfio devices")
	}
	return pciDeviceMap
}

func (c *DeviceController) buildConfiguredDeviceMap() map[string]string {
	resources := c.resourceConfig.GetResources()
	devicesMap := make(map[string]string)
	for _, resource := range resources {
		for _, address := range resource.Addresses {
			devicesMap[address] = resource.Name
		}
	}
	return devicesMap
}

func (c *DeviceController) startDevice(resourceName string, dev Device) {
	c.stopDevice(resourceName)
	controlledDev := controlledDevice{
		devicePlugin: dev,
		backoff:      c.backoff,
	}
	controlledDev.Start()
	c.startedPlugins[resourceName] = controlledDev
}

func (c *DeviceController) stopDevice(resourceName string) {
	dev, exists := c.startedPlugins[resourceName]
	if exists {
		dev.Stop()
		delete(c.startedPlugins, resourceName)
	}
}
