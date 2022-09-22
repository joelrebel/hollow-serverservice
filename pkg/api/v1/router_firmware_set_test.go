package serverservice_test

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	serverservice "go.hollow.sh/serverservice/pkg/api/v1"
)

// r640FirmwareFixtureUUIDs returns firmware  uuids based on the firmware hardware model attribute
func r640FirmwareFixtureUUIDs(t *testing.T, firmware []serverservice.ComponentFirmwareVersion) []string {
	t.Helper()

	ids := []string{}

	for idx, f := range firmware {
		if f.Model == "R640" {
			ids = append(ids, firmware[idx].UUID.String())
		}
	}

	return ids
}

func TestIntegrationServerComponentFirmwareSetCreate(t *testing.T) {
	s := serverTest(t)

	var r640FirmwareIDs []string

	realClientTests(t, func(ctx context.Context, authToken string, respCode int, expectError bool) error {
		s.Client.SetToken(authToken)

		var testFirmwareSet serverservice.ComponentFirmwareSetPayload

		if !expectError {
			// 2. retrieve component firmware fixture data for test
			firmware, _, err := s.Client.ListServerComponentFirmware(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}

			r640FirmwareIDs = r640FirmwareFixtureUUIDs(t, firmware)

			// expect two fixture firmware objects to be returned
			assert.Equal(t, 2, len(r640FirmwareIDs))

			testFirmwareSet = serverservice.ComponentFirmwareSetPayload{
				Name:                   "test-firmware-set",
				ComponentFirmwareUUIDs: r640FirmwareIDs,
				Metadata:               []byte(`{"created by": "foobar"}`),
			}
		}

		id, resp, err := s.Client.CreateServerComponentFirmwareSet(ctx, testFirmwareSet)
		if !expectError {
			require.NoError(t, err)
			assert.NotNil(t, id)
			assert.Equal(t, "resource created", resp.Message)
			assert.NotNil(t, resp.Links.Self)
		}

		return err
	})

	var testCases = []struct {
		testName           string
		firmwareSetPayload *serverservice.ComponentFirmwareSetPayload
		expectedError      bool
		expectedResponse   string
		errorMsg           string
	}{

		{
			"Name field required",
			&serverservice.ComponentFirmwareSetPayload{},
			true,
			"400",
			"Error:Field validation for 'Name' failed on the 'required' tag",
		},
		{
			"component firmware UUIDs required",
			&serverservice.ComponentFirmwareSetPayload{
				Name: "foobar",
			},
			true,
			"400",
			"expected one or more firmware UUIDs, got none",
		},
		{
			"valid firmware UUIDs expected",
			&serverservice.ComponentFirmwareSetPayload{
				Name: "foobar",
				ComponentFirmwareUUIDs: []string{
					r640FirmwareIDs[0],
					"d825bbeb-20fb-452e-9fe4-invalid",
				},
			},
			true,
			"400",
			"invalid firmware UUID",
		},
		{
			"pre-existing firmware UUIDs expected for firmware set",
			&serverservice.ComponentFirmwareSetPayload{
				Name: "foobar",
				ComponentFirmwareUUIDs: []string{
					"d825bbeb-20fb-452e-9fe4-cdedacb2ca1f",
				},
			},
			true,
			"400",
			"firmware UUID does not exist",
		},
		{
			"pre-existing firmware UUIDs expected for firmware set",
			&serverservice.ComponentFirmwareSetPayload{
				Name:                   "foobar",
				ComponentFirmwareUUIDs: r640FirmwareIDs,
			},
			true,
			"400",
			"firmware UUID does not exist",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			fwUUID, resp, err := s.Client.CreateServerComponentFirmwareSet(context.TODO(), *tt.firmwareSetPayload)
			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Contains(t, err.Error(), tt.expectedResponse)
				return
			}

			spew.Dump(resp)
			assert.NotEqual(t, uuid.Nil, fwUUID)
		})
	}
}
