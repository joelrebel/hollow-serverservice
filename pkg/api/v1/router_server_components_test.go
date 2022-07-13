package serverservice_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.hollow.sh/serverservice/internal/dbtools"
	serverservice "go.hollow.sh/serverservice/pkg/api/v1"
)

// zero values that change for each test run to enable object comparison
func zeroUUIDValues(sc *serverservice.ServerComponent) {
	sc.ServerUUID = uuid.UUID{}
	sc.UUID = uuid.UUID{}
	sc.ComponentTypeID = ""
}

func zeroTimeValues(sc *serverservice.ServerComponent) {
	sc.CreatedAt = time.Time{}
	sc.UpdatedAt = time.Time{}

	for i := 0; i < len(sc.VersionedAttributes); i++ {
		sc.VersionedAttributes[i].CreatedAt = time.Time{}
		sc.VersionedAttributes[i].LastReportedAt = time.Time{}
	}
}

func componentByNameVendorModelSerial(name, vendor, model, serial string, sc serverservice.ServerComponentSlice) *serverservice.ServerComponent {
	for _, c := range sc {
		if c.Name == name && c.Vendor == vendor && c.Model == model && c.Serial == serial {
			return &c
		}
	}

	return nil
}

func TestIntegrationServerListComponents(t *testing.T) {
	s := serverTest(t)

	realClientTests(t, func(ctx context.Context, authToken string, respCode int, expectError bool) error {
		s.Client.SetToken(authToken)

		attrs, _, err := s.Client.ListComponents(ctx, nil)
		if !expectError {
			require.NoError(t, err)
			assert.Len(t, attrs, 7)
		}

		return err
	})

	testCases := []struct {
		testName string
		params   *serverservice.ServerComponentListParams
		expected serverservice.ServerComponentSlice
		errorMsg string
	}{
		// TODO(joel): tests for unhappy paths
		{
			"by model",
			&serverservice.ServerComponentListParams{Model: "Belly"},
			serverservice.ServerComponentSlice{
				{
					Model:             "Belly",
					Serial:            "Up",
					ComponentTypeName: "Fins",
					ComponentTypeSlug: "fins",
				},
			},
			"",
		},
		{
			"by model, versioned attributes",
			&serverservice.ServerComponentListParams{
				Model: "Normal Fin",
				VersionedAttributeListParams: []serverservice.AttributeListParams{
					{
						Namespace: "hollow.versioned",
						Keys:      []string{"something"},
						Operator:  "eq",
						Value:     "cool",
					},
				},
			},
			serverservice.ServerComponentSlice{
				{
					Model:             "Normal Fin",
					Serial:            "Left",
					Name:              "Normal Fin",
					ComponentTypeName: "Fins",
					ComponentTypeSlug: "fins",
					VersionedAttributes: []serverservice.VersionedAttributes{
						{
							Namespace: "hollow.versioned",
							Data:      json.RawMessage(`{"something":"cool"}`),
						},
					},
				},
			},
			"",
		},
		{
			"pagination Limit",
			&serverservice.ServerComponentListParams{
				Pagination: &serverservice.PaginationParams{
					Limit: 1,
				},
				Model: "Belly",
			},
			serverservice.ServerComponentSlice{
				{
					Model:             "Belly",
					Serial:            "Up",
					ComponentTypeName: "Fins",
					ComponentTypeSlug: "fins",
				},
			},
			"",
		},
		{
			"pagination Limit, Offset",
			&serverservice.ServerComponentListParams{
				Pagination: &serverservice.PaginationParams{
					Limit: 1,
					Page:  2,
				},
			},
			serverservice.ServerComponentSlice{
				{
					Name:              "Normal Fin",
					Serial:            "Right",
					ComponentTypeName: "Fins",
					ComponentTypeSlug: "fins",
				},
			},
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			got, res, err := s.Client.ListComponents(context.TODO(), tc.params)
			if tc.errorMsg != "" {
				assert.Contains(t, err.Error(), tc.errorMsg)
				return
			}

			assert.Nil(t, err)
			assert.NotNil(t, res)

			// zero timestamp, uuid values for object comparison
			for i := 0; i < len(got); i++ {
				zeroUUIDValues(&got[i])
				zeroTimeValues(&got[i])
			}

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestIntegrationServerGetComponents(t *testing.T) {
	s := serverTest(t)

	realClientTests(t, func(ctx context.Context, authToken string, respCode int, expectError bool) error {
		s.Client.SetToken(authToken)

		attrs, _, err := s.Client.GetComponents(ctx, uuid.MustParse(dbtools.FixtureNemo.ID), nil)
		if !expectError {
			require.NoError(t, err)
			assert.Len(t, attrs, 2)
		}

		return err
	})

	// init fixture data

	// 1. get list of servers
	servers, _, err := s.Client.List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// expect atleast 1 server for test to proceed
	assert.GreaterOrEqual(t, len(servers), 1)

	// 2. get component type slice
	componentTypeSlice, _, err := s.Client.ListServerComponentTypes(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// expect atleast 1 component type to proceed
	assert.Len(t, componentTypeSlice, 1)

	// fixture to create a server components
	csFixtureCreate := serverservice.ServerComponentSlice{
		{
			ServerUUID:        servers[0].UUID,
			Name:              "My Lucky Fin",
			Vendor:            "barracuda",
			Model:             "a lucky fin",
			Serial:            "right",
			ComponentTypeID:   componentTypeSlice.ByName("Fins").ID,
			ComponentTypeName: componentTypeSlice.ByName("Fins").Name,
			ComponentTypeSlug: componentTypeSlice.ByName("Fins").Slug,
			VersionedAttributes: []serverservice.VersionedAttributes{
				{
					Namespace: dbtools.FixtureNamespaceVersioned,
					Data:      json.RawMessage(`{"version":"1.0"}`),
				},
				{
					Namespace: dbtools.FixtureNamespaceVersioned,
					Data:      json.RawMessage(`{"version":"2.0"}`),
				},
			},
		},
	}

	// create server component
	_, err = s.Client.CreateComponents(context.TODO(), servers[0].UUID, csFixtureCreate)
	if err != nil {
		t.Fatal(err)
	}

	var testCases = []struct {
		testName        string
		srvUUID         uuid.UUID
		expectedCount   int
		expectedInSlice serverservice.ServerComponent
		errorMsg        string
	}{
		{
			"returns not found on missing server uuid",
			uuid.New(),
			0,
			serverservice.ServerComponent{},
			"response code: 404",
		},
		{
			"component Versioned Attributes is returned as expected",
			servers[0].UUID,
			3,
			serverservice.ServerComponent{
				ServerUUID:        servers[0].UUID,
				Name:              "My Lucky Fin",
				Vendor:            "barracuda",
				Model:             "a lucky fin",
				Serial:            "right",
				ComponentTypeID:   componentTypeSlice.ByName("Fins").ID,
				ComponentTypeName: componentTypeSlice.ByName("Fins").Name,
				ComponentTypeSlug: componentTypeSlice.ByName("Fins").Slug,
				VersionedAttributes: []serverservice.VersionedAttributes{
					{
						Namespace: dbtools.FixtureNamespaceVersioned,
						Data:      json.RawMessage(`{"version":"2.0"}`),
					},
				},
			},
			"",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			got, _, err := s.Client.GetComponents(context.TODO(), tt.srvUUID, nil)
			if tt.errorMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			assert.Nil(t, err)

			assert.Equal(t, tt.expectedCount, len(got))
			gotc := componentByNameVendorModelSerial(
				tt.expectedInSlice.Name,
				tt.expectedInSlice.Vendor,
				tt.expectedInSlice.Model,
				tt.expectedInSlice.Serial,
				got,
			)

			if gotc == nil {
				t.Fatal("expected component, got nil")
			}

			// zero variable values before comparison
			gotc.UUID = uuid.Nil
			zeroTimeValues(gotc)

			assert.Equal(t, tt.expectedInSlice, *gotc)
		})
	}
}

func TestIntegrationServerCreateComponents(t *testing.T) {
	s := serverTest(t)

	// fixture objects
	var servers []serverservice.Server

	var componentTypeSlice serverservice.ServerComponentTypeSlice

	// run default client tests
	realClientTests(t, func(ctx context.Context, authToken string, respCode int, expectError bool) error {
		s.Client.SetToken(authToken)

		var sc serverservice.ServerComponentSlice

		if !expectError {
			var err error
			// 2. retrieve list of servers, expect the test db is populated with one or more test servers
			servers, _, err = s.Client.List(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}

			componentTypeSlice, _, err = s.Client.ListServerComponentTypes(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}

			sc = serverservice.ServerComponentSlice{
				{
					ServerUUID:        servers[0].UUID,
					ComponentTypeID:   componentTypeSlice[0].ID,
					ComponentTypeName: componentTypeSlice[0].Name,
					ComponentTypeSlug: componentTypeSlice[0].Slug,
					Name:              "Fin A",
					Model:             "Normal Fin",
					Serial:            "Left Upper",
				},
			}
		}

		_, err := s.Client.CreateComponents(ctx, uuid.MustParse(dbtools.FixtureNemo.ID), sc)
		if !expectError {
			require.NoError(t, err)
		}

		return err
	})

	// make sure all fixtures are available
	assert.GreaterOrEqual(t, len(servers), 1)
	assert.GreaterOrEqual(t, len(componentTypeSlice), 1)

	var testCases = []struct {
		testName    string
		serverUUID  uuid.UUID
		components  serverservice.ServerComponentSlice
		responseMsg string
		errorMsg    string
	}{
		{
			"unknown server query returns 404",
			uuid.New(),
			nil,
			"",
			"hollow client received a server error - response code: 404, message: resource not found",
		},
		{
			"create component and list by Name works",
			servers[0].UUID,
			serverservice.ServerComponentSlice{
				{
					ServerUUID:        servers[0].UUID,
					ComponentTypeID:   componentTypeSlice[0].ID,
					ComponentTypeName: componentTypeSlice[0].Name,
					ComponentTypeSlug: componentTypeSlice[0].Slug,
					Name:              "Fin B",
					Model:             "Normal Fin",
					Serial:            "Left Lower",
				},
			},
			"resource created",
			"",
		},
		{
			"create component which violates unique constraint on ServerID, ComponentTypeID, Serial should return error",
			servers[0].UUID,
			serverservice.ServerComponentSlice{
				{
					ServerUUID:        servers[0].UUID,
					ComponentTypeID:   componentTypeSlice[0].ID,
					ComponentTypeName: componentTypeSlice[0].Name,
					ComponentTypeSlug: componentTypeSlice[0].Slug,
					Name:              "Fin B",
					Model:             "Normal Fin",
					Serial:            "Left Lower",
				},
				{
					ServerUUID:        servers[0].UUID,
					ComponentTypeID:   componentTypeSlice[0].ID,
					ComponentTypeName: componentTypeSlice[0].Name,
					ComponentTypeSlug: componentTypeSlice[0].Slug,
					Name:              "Fin B",
					Model:             "Normal Fin",
					Serial:            "Left Lower",
				},
			},
			"",
			"duplicate key value violates unique constraint",
		},
		{
			"create component with unknown server UUID returns error",
			uuid.New(),
			serverservice.ServerComponentSlice{
				{
					ServerUUID:        uuid.New(),
					ComponentTypeID:   componentTypeSlice[0].ID,
					ComponentTypeName: componentTypeSlice[0].Name,
					ComponentTypeSlug: componentTypeSlice[0].Slug,
					Name:              "Fin B",
					Model:             "Normal Fin",
					Serial:            "Left Lower 2",
				},
			},
			"",
			"resource not found",
		},
		{
			"create component validates field constraints",
			servers[0].UUID,
			serverservice.ServerComponentSlice{
				{
					ServerUUID:      servers[0].UUID,
					ComponentTypeID: "lala",
					Name:            "Fin B",
					Model:           "Normal Fin",
					Serial:          "Left Lower 2",
				},
			},
			"",
			"error in server component payload",
		},
		{
			"create component with empty slice returns error",
			servers[0].UUID,
			serverservice.ServerComponentSlice{},
			"",
			"error in server component payload",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			// create
			res, err := s.Client.CreateComponents(context.TODO(), tt.serverUUID, tt.components)
			if tt.errorMsg != "" {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			assert.Nil(t, err)
			assert.NotNil(t, res)
			assert.Contains(t, res.Message, tt.responseMsg)

			// list
			params := &serverservice.ServerComponentListParams{Model: "a lucky fin"}
			got, _, err := s.Client.ListComponents(context.TODO(), params)
			if err != nil {
				t.Error(err)
			}

			// zero timestamp, uuid values for object comparison
			for i := 0; i < len(got); i++ {
				zeroTimeValues(&got[i])
			}

			// compare object in fixture with object from store
			assert.ElementsMatch(t, fixture, got)
		})
	}
}

func TestIntegrationServerUpdateComponents(t *testing.T) {
	s := serverTest(t)

	// run default client tests
	realClientTests(t, func(ctx context.Context, authToken string, respCode int, expectError bool) error {
		s.Client.SetToken(authToken)

		_, err := s.Client.UpdateComponents(ctx, uuid.MustParse(dbtools.FixtureNemo.ID), nil)
		if !expectError {
			require.NoError(t, err)
		}

		return err
	})

	// init fixture data
	//
	// 1. retrieve list of component types, expect the test db is populated with the 'Fins' component type information
	componentTypeSlice, _, err := s.Client.ListServerComponentTypes(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, componentTypeSlice, 1)

	// 2. retrieve list of servers, expect the test db is populated with one or more test servers
	servers, _, err := s.Client.List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.GreaterOrEqual(t, len(servers), 1)

	// 3. declare fixture
	fixture := serverservice.ServerComponentSlice{
		{
			UUID:              uuid.New(),
			ServerUUID:        servers[0].UUID,
			Name:              "My Lucky Fin",
			Vendor:            "barracuda",
			Model:             "a lucky fin",
			Serial:            "right",
			ComponentTypeID:   componentTypeSlice.ByName("Fins").ID,
			ComponentTypeName: componentTypeSlice.ByName("Fins").Name,
			ComponentTypeSlug: componentTypeSlice.ByName("Fins").Slug,
			VersionedAttributes: []serverservice.VersionedAttributes{
				{
					Namespace: dbtools.FixtureNamespaceVersioned,
					Data:      json.RawMessage(`{"version":"1.0"}`),
				},
			},
		},
	}

	var testCases = []struct {
		testName                   string
		serverUUID                 uuid.UUID
		components                 serverservice.ServerComponentSlice
		updatedName                string
		updatedVersionedAttributes json.RawMessage
		responseMsg                string
		errorMsg                   string
	}{
		{
			"unknown server query returns 404",
			uuid.New(),
			nil,
			"",
			nil,
			"",
			"hollow client received a server error - response code: 404, message: resource not found",
		},
		{
			"create, update and query by model",
			servers[0].UUID,
			fixture,
			"renamed",
			json.RawMessage(`{"version":"1.1"}`),
			"resource updated",
			"",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			// create
			res, err := s.Client.CreateComponents(context.TODO(), tt.serverUUID, tt.components)
			if tt.errorMsg != "" {
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			assert.Nil(t, err)
			assert.NotNil(t, res)
			assert.Contains(t, res.Message, "resource created")

			fixture[0].Name = tt.updatedName
			fixture[0].VersionedAttributes[0].Data = tt.updatedVersionedAttributes

			// update
			res, err = s.Client.UpdateComponents(context.TODO(), tt.serverUUID, tt.components)
			if tt.errorMsg != "" {
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			assert.Contains(t, res.Message, tt.responseMsg)

			// list
			params := &serverservice.ServerComponentListParams{Model: "a lucky fin"}
			got, _, err := s.Client.ListComponents(context.TODO(), params)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.updatedName, got[0].Name)
			assert.Equal(t, tt.updatedVersionedAttributes, got[0].VersionedAttributes[0].Data)

			// zero timestamp, uuid values for object comparison
			for i := 0; i < len(got); i++ {
				zeroTimeValues(&got[i])
			}

			// compare object in fixture with object from store
			assert.ElementsMatch(t, fixture, got)
		})
	}
}
