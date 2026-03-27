package switchbotcli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	switchbot "github.com/yasu89/switch-bot-api-go"
)

func TestNewClientClonesAliases(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{
		"DIY Light": "Light",
	}
	client := NewClient("token", "secret", "", aliases)

	aliases["DIY Light"] = "TV"

	normalized := client.normalizeDevice(&switchbot.InfraredRemoteDevice{
		DeviceID:    "device-id",
		DeviceName:  "Living Room",
		RemoteType:  "DIY Light",
		HubDeviceId: "hub-id",
	})

	assert.IsType(t, &switchbot.InfraredRemoteLightDevice{}, normalized)
}

func TestNormalizeDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		aliases    map[string]string
		remoteType string
		wantType   any
	}{
		{
			name:       "keeps custom type generic without aliases",
			aliases:    nil,
			remoteType: "DIY Light",
			wantType:   &switchbot.InfraredRemoteDevice{},
		},
		{
			name:       "maps configured aliases",
			aliases:    map[string]string{"Custom TV": "TV"},
			remoteType: "Custom TV",
			wantType:   &switchbot.InfraredRemoteTVDevice{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient("token", "secret", "", tt.aliases)
			device := &switchbot.InfraredRemoteDevice{
				DeviceID:    "device-id",
				DeviceName:  "Living Room",
				RemoteType:  tt.remoteType,
				HubDeviceId: "hub-id",
			}

			normalized := client.normalizeDevice(device)

			assert.IsType(t, tt.wantType, normalized)
			assert.Equal(t, "device-id", normalized.(switchbot.DeviceIDGettable).GetDeviceID())
		})
	}
}

func TestWrapInfraredRemoteDeviceSupportedTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		normalizedType string
		wantType       any
	}{
		{name: "air conditioner", normalizedType: "Air Conditioner", wantType: &switchbot.InfraredRemoteAirConditionerDevice{}},
		{name: "tv", normalizedType: "TV", wantType: &switchbot.InfraredRemoteTVDevice{}},
		{name: "light", normalizedType: "Light", wantType: &switchbot.InfraredRemoteLightDevice{}},
		{name: "streamer", normalizedType: "Streamer", wantType: &switchbot.InfraredRemoteStreamerDevice{}},
		{name: "set top box", normalizedType: "Set Top Box", wantType: &switchbot.InfraredRemoteSetTopBoxDevice{}},
		{name: "dvd player", normalizedType: "DVD Player", wantType: &switchbot.InfraredRemoteDvdPlayerDevice{}},
		{name: "fan", normalizedType: "Fan", wantType: &switchbot.InfraredRemoteFanDevice{}},
		{name: "projector", normalizedType: "Projector", wantType: &switchbot.InfraredRemoteProjectorDevice{}},
		{name: "camera", normalizedType: "Camera", wantType: &switchbot.InfraredRemoteCameraDevice{}},
		{name: "air purifier", normalizedType: "Air Purifier", wantType: &switchbot.InfraredRemoteAirPurifierDevice{}},
		{name: "speaker", normalizedType: "Speaker", wantType: &switchbot.InfraredRemoteSpeakerDevice{}},
		{name: "water heater", normalizedType: "Water Heater", wantType: &switchbot.InfraredRemoteWaterHeaterDevice{}},
		{name: "robot vacuum cleaner", normalizedType: "Robot Vacuum Cleaner", wantType: &switchbot.InfraredRemoteRobotVacuumCleanerDevice{}},
		{name: "others", normalizedType: "Others", wantType: &switchbot.InfraredRemoteOthersDevice{}},
		{name: "unknown stays generic", normalizedType: "Custom", wantType: &switchbot.InfraredRemoteDevice{}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			device := &switchbot.InfraredRemoteDevice{
				DeviceID:    "device-id",
				DeviceName:  "Living Room",
				RemoteType:  "source-type",
				HubDeviceId: "hub-id",
			}

			wrapped := wrapInfraredRemoteDevice(device, tt.normalizedType)

			assert.IsType(t, tt.wantType, wrapped)
			assert.Equal(t, "device-id", wrapped.(switchbot.DeviceIDGettable).GetDeviceID())
		})
	}
}

func TestListDevicesEnrichesOutputAndAppliesAliases(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/devices", r.URL.Path)

		writeJSONResponse(t, w, map[string]any{
			"statusCode": 100,
			"body": map[string]any{
				"deviceList": []any{
					map[string]any{
						"deviceId":           "bot-1",
						"deviceType":         "Bot",
						"hubDeviceId":        "hub-1",
						"deviceName":         "Desk Switch",
						"enableCloudService": true,
					},
				},
				"infraredRemoteList": []any{
					map[string]any{
						"deviceId":    "ir-light-1",
						"deviceName":  "Living Room Light",
						"remoteType":  "DIY Light",
						"hubDeviceId": "hub-1",
					},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	client := NewClient("token", "secret", server.URL, map[string]string{
		"DIY Light": "Light",
	})

	output, err := client.ListDevices()
	require.NoError(t, err)

	require.Len(t, output.DeviceList, 1)
	assert.Equal(t, "Bot", output.DeviceList[0]["deviceType"])
	assert.NotContains(t, output.DeviceList[0], "Client")
	assert.Contains(t, output.DeviceList[0], "commandParameterJSONSchema")

	require.Len(t, output.InfraredRemoteList, 1)
	assert.Equal(t, "DIY Light", output.InfraredRemoteList[0]["remoteType"])
	assert.NotContains(t, output.InfraredRemoteList[0], "Client")
	assert.Contains(t, output.InfraredRemoteList[0], "commandParameterJSONSchema")
}

func TestGetStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns device status", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/devices":
				writeJSONResponse(t, w, map[string]any{
					"statusCode": 100,
					"body": map[string]any{
						"deviceList": []any{
							map[string]any{
								"deviceId":           "bot-1",
								"deviceType":         "Bot",
								"hubDeviceId":        "hub-1",
								"deviceName":         "Desk Switch",
								"enableCloudService": true,
							},
						},
						"infraredRemoteList": []any{},
					},
				})
			case r.Method == http.MethodGet && r.URL.Path == "/devices/bot-1/status":
				writeJSONResponse(t, w, map[string]any{
					"statusCode": 100,
					"body": map[string]any{
						"deviceId":    "bot-1",
						"deviceType":  "Bot",
						"hubDeviceId": "hub-1",
						"power":       "on",
						"battery":     88,
						"version":     "V1.0",
						"deviceMode":  "switchMode",
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(server.Close)

		client := NewClient("token", "secret", server.URL, nil)

		status, err := client.GetStatus("bot-1")
		require.NoError(t, err)

		botStatus, ok := status.(*switchbot.BotDeviceStatusBody)
		require.True(t, ok)
		assert.Equal(t, "on", botStatus.Power)
		assert.Equal(t, 88, botStatus.Battery)
	})

	t.Run("returns not found for unknown device", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSONResponse(t, w, map[string]any{
				"statusCode": 100,
				"body": map[string]any{
					"deviceList":         []any{},
					"infraredRemoteList": []any{},
				},
			})
		}))
		t.Cleanup(server.Close)

		client := NewClient("token", "secret", server.URL, nil)

		_, err := client.GetStatus("missing")
		require.ErrorIs(t, err, ErrDeviceNotFound)
	})
}

func TestExecuteCommandUsesInfraredAliases(t *testing.T) {
	t.Parallel()

	var gotRequest map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/devices":
			writeJSONResponse(t, w, map[string]any{
				"statusCode": 100,
				"body": map[string]any{
					"deviceList": []any{},
					"infraredRemoteList": []any{
						map[string]any{
							"deviceId":    "ir-light-1",
							"deviceName":  "Living Room Light",
							"remoteType":  "DIY Light",
							"hubDeviceId": "hub-1",
						},
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/devices/ir-light-1/commands":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotRequest))
			writeJSONResponse(t, w, map[string]any{
				"statusCode": 100,
				"message":    "success",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient("token", "secret", server.URL, map[string]string{
		"DIY Light": "Light",
	})

	response, err := client.ExecuteCommand("ir-light-1", `{"command":"TurnOn"}`)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"command":     "turnOn",
		"commandType": "command",
		"parameter":   "default",
	}, gotRequest)
	assert.Equal(t, 100, response.StatusCode)
	assert.Equal(t, "success", response.Message)
}

func TestExecuteCommandReturnsNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONResponse(t, w, map[string]any{
			"statusCode": 100,
			"body": map[string]any{
				"deviceList":         []any{},
				"infraredRemoteList": []any{},
			},
		})
	}))
	t.Cleanup(server.Close)

	client := NewClient("token", "secret", server.URL, nil)

	_, err := client.ExecuteCommand("missing", `{"command":"TurnOn"}`)
	require.ErrorIs(t, err, ErrDeviceNotFound)
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, payload map[string]any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}
