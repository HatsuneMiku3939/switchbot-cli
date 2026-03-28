package switchbotcli

import (
	"encoding/json"
	"errors"

	switchbot "github.com/yasu89/switch-bot-api-go"
)

var ErrDeviceNotFound = errors.New("device not found")

type DevicesOutput struct {
	DeviceList         []map[string]interface{} `json:"deviceList"`
	InfraredRemoteList []map[string]interface{} `json:"infraredRemoteList"`
}

type Client struct {
	apiClient                 *switchbot.Client
	infraredRemoteTypeAliases map[string]string
}

func NewClient(token string, secret string, baseURL string, infraredRemoteTypeAliases map[string]string) *Client {
	options := []switchbot.Option{}
	if baseURL != "" {
		options = append(options, switchbot.OptionBaseApiURL(baseURL))
	}

	return &Client{
		apiClient:                 switchbot.NewClient(secret, token, options...),
		infraredRemoteTypeAliases: cloneStringMap(infraredRemoteTypeAliases),
	}
}

func (c *Client) ListDevices() (*DevicesOutput, error) {
	response, err := c.apiClient.GetDevices()
	if err != nil {
		return nil, err
	}

	result := &DevicesOutput{}

	for _, device := range response.Body.DeviceList {
		deviceMap, err := enrichDevice(c.normalizeDevice(device))
		if err != nil {
			return nil, err
		}
		result.DeviceList = append(result.DeviceList, deviceMap)
	}

	for _, device := range response.Body.InfraredRemoteList {
		deviceMap, err := enrichDevice(c.normalizeDevice(device))
		if err != nil {
			return nil, err
		}
		result.InfraredRemoteList = append(result.InfraredRemoteList, deviceMap)
	}

	return result, nil
}

func (c *Client) GetStatus(deviceID string) (interface{}, error) {
	response, err := c.apiClient.GetDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range response.Body.DeviceList {
		device = c.normalizeDevice(device)

		statusDevice, ok := device.(switchbot.StatusGettable)
		if !ok {
			continue
		}

		deviceIDGettable, ok := device.(switchbot.DeviceIDGettable)
		if !ok || deviceIDGettable.GetDeviceID() != deviceID {
			continue
		}

		return statusDevice.GetAnyStatusBody()
	}

	return nil, ErrDeviceNotFound
}

func (c *Client) ExecuteCommand(deviceID string, commandParameterJSON string) (*switchbot.CommonResponse, error) {
	response, err := c.apiClient.GetDevices()
	if err != nil {
		return nil, err
	}

	targetDevice, err := findExecutableDeviceWithNormalizer(response.Body.DeviceList, deviceID, c.normalizeDevice)
	if err == nil {
		return targetDevice.ExecCommand(commandParameterJSON)
	}
	if !errors.Is(err, ErrDeviceNotFound) {
		return nil, err
	}

	targetDevice, err = findExecutableDeviceWithNormalizer(response.Body.InfraredRemoteList, deviceID, c.normalizeDevice)
	if err != nil {
		return nil, err
	}

	return targetDevice.ExecCommand(commandParameterJSON)
}

func findExecutableDevice(devices []interface{}, deviceID string) (switchbot.ExecutableCommandDevice, error) {
	return findExecutableDeviceWithNormalizer(devices, deviceID, func(device interface{}) interface{} {
		return device
	})
}

func findExecutableDeviceWithNormalizer(
	devices []interface{},
	deviceID string,
	normalize func(device interface{}) interface{},
) (switchbot.ExecutableCommandDevice, error) {
	for _, device := range devices {
		device = normalize(device)

		executable, ok := device.(switchbot.ExecutableCommandDevice)
		if !ok {
			continue
		}

		deviceIDGettable, ok := device.(switchbot.DeviceIDGettable)
		if !ok || deviceIDGettable.GetDeviceID() != deviceID {
			continue
		}

		return executable, nil
	}

	return nil, ErrDeviceNotFound
}

func (c *Client) normalizeDevice(device interface{}) interface{} {
	infraredDevice, ok := device.(*switchbot.InfraredRemoteDevice)
	if !ok {
		return device
	}

	normalizedType := infraredDevice.RemoteType
	if alias, ok := c.infraredRemoteTypeAliases[infraredDevice.RemoteType]; ok {
		normalizedType = alias
	}

	return wrapInfraredRemoteDevice(infraredDevice, normalizedType)
}

func wrapInfraredRemoteDevice(device *switchbot.InfraredRemoteDevice, normalizedType string) interface{} {
	switch normalizedType {
	case "Air Conditioner":
		return &switchbot.InfraredRemoteAirConditionerDevice{
			InfraredRemoteDevice: *device,
		}
	case "TV":
		return &switchbot.InfraredRemoteTVDevice{
			InfraredRemoteDevice: *device,
		}
	case "Light":
		return &switchbot.InfraredRemoteLightDevice{
			InfraredRemoteDevice: *device,
		}
	case "Streamer":
		return &switchbot.InfraredRemoteStreamerDevice{
			InfraredRemoteTVDevice: switchbot.InfraredRemoteTVDevice{
				InfraredRemoteDevice: *device,
			},
		}
	case "Set Top Box":
		return &switchbot.InfraredRemoteSetTopBoxDevice{
			InfraredRemoteTVDevice: switchbot.InfraredRemoteTVDevice{
				InfraredRemoteDevice: *device,
			},
		}
	case "DVD Player":
		return &switchbot.InfraredRemoteDvdPlayerDevice{
			InfraredRemoteDevice: *device,
		}
	case "Fan":
		return &switchbot.InfraredRemoteFanDevice{
			InfraredRemoteDevice: *device,
		}
	case "Projector":
		return &switchbot.InfraredRemoteProjectorDevice{
			InfraredRemoteDevice: *device,
		}
	case "Camera":
		return &switchbot.InfraredRemoteCameraDevice{
			InfraredRemoteDevice: *device,
		}
	case "Air Purifier":
		return &switchbot.InfraredRemoteAirPurifierDevice{
			InfraredRemoteDevice: *device,
		}
	case "Speaker":
		return &switchbot.InfraredRemoteSpeakerDevice{
			InfraredRemoteDvdPlayerDevice: switchbot.InfraredRemoteDvdPlayerDevice{
				InfraredRemoteDevice: *device,
			},
		}
	case "Water Heater":
		return &switchbot.InfraredRemoteWaterHeaterDevice{
			InfraredRemoteDevice: *device,
		}
	case "Robot Vacuum Cleaner":
		return &switchbot.InfraredRemoteRobotVacuumCleanerDevice{
			InfraredRemoteDevice: *device,
		}
	case "Others":
		return &switchbot.InfraredRemoteOthersDevice{
			Client:      device.Client,
			DeviceID:    device.DeviceID,
			DeviceName:  device.DeviceName,
			RemoteType:  device.RemoteType,
			HubDeviceId: device.HubDeviceId,
		}
	default:
		return device
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}

	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}

	return cloned
}

func enrichDevice(device interface{}) (map[string]interface{}, error) {
	deviceJSON, err := json.Marshal(device)
	if err != nil {
		return nil, err
	}

	var deviceMap map[string]interface{}
	if err := json.Unmarshal(deviceJSON, &deviceMap); err != nil {
		return nil, err
	}

	// Drop library-internal fields from CLI output.
	delete(deviceMap, "Client")

	if executable, ok := device.(switchbot.ExecutableCommandDevice); ok {
		schema, err := executable.GetCommandParameterJSONSchema()
		if err != nil {
			return nil, err
		}

		var parsedSchema interface{}
		if err := json.Unmarshal([]byte(schema), &parsedSchema); err != nil {
			return nil, err
		}

		deviceMap["commandParameterJSONSchema"] = parsedSchema
	}

	return deviceMap, nil
}
