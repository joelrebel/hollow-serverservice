package serverservice

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"

	"go.hollow.sh/serverservice/internal/models"
)

// Server represents a server in a facility
type Server struct {
	UUID                uuid.UUID             `json:"uuid"`
	Name                string                `json:"name"`
	FacilityCode        string                `json:"facility"`
	FirmwareSetUUID     uuid.UUID             `json:"firmware_set_uuid"`
	Attributes          []Attributes          `json:"attributes"`
	Components          []ServerComponent     `json:"components"`
	VersionedAttributes []VersionedAttributes `json:"versioned_attributes"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
	// DeletedAt is a pointer to a Time in order to be able to support checks for nil time
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (r *Router) getServers(c *gin.Context, params ServerListParams) (models.ServerSlice, int64, error) {
	mods := params.queryMods()

	count, err := models.Servers(mods...).Count(c.Request.Context(), r.DB)
	if err != nil {
		return nil, 0, err
	}

	// add pagination
	params.PaginationParams.Preload = true
	params.PaginationParams.OrderBy = models.ServerTableColumns.CreatedAt + " DESC"
	mods = append(mods, params.PaginationParams.serverQueryMods()...)

	s, err := models.Servers(mods...).All(c.Request.Context(), r.DB)
	if err != nil {
		return s, 0, err
	}

	return s, count, nil
}

func (s *Server) fromDBModel(dbS *models.Server) error {
	var err error

	s.UUID, err = uuid.Parse(dbS.ID)
	if err != nil {
		return err
	}

	firmwareIDStr := null.String(dbS.FirmwareSetID).String
	if firmwareIDStr != "" {
		s.FirmwareSetUUID, err = uuid.Parse(firmwareIDStr)
		if err != nil {
			return err
		}
	}

	s.Name = dbS.Name.String
	s.FacilityCode = dbS.FacilityCode.String
	s.CreatedAt = dbS.CreatedAt.Time
	s.UpdatedAt = dbS.UpdatedAt.Time

	if !dbS.DeletedAt.IsZero() {
		s.DeletedAt = &dbS.DeletedAt.Time
	}

	if dbS.R != nil {
		if dbS.R.Attributes != nil {
			s.Attributes, err = convertFromDBAttributes(dbS.R.Attributes)
			if err != nil {
				return err
			}
		}

		if dbS.R.ServerComponents != nil {
			s.Components, err = convertDBServerComponents(dbS.R.ServerComponents)
			if err != nil {
				return err
			}
		}

		if dbS.R.VersionedAttributes != nil {
			s.VersionedAttributes, err = convertFromDBVersionedAttributes(dbS.R.VersionedAttributes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Server) toDBModel() (*models.Server, error) {
	dbS := &models.Server{
		Name:          null.StringFrom(s.Name),
		FacilityCode:  null.StringFrom(s.FacilityCode),
		FirmwareSetID: null.StringFrom(s.FirmwareSetUUID.String()),
	}

	if s.UUID.String() != uuid.Nil.String() {
		dbS.ID = s.UUID.String()
	}

	dbS.FirmwareSetID = nullUUID(s.FirmwareSetUUID)

	return dbS, nil
}

// nullUUID returns a null.String type from a uuid.UUID.
//
// in the case where the uuid is nil, a new null.String{} object is returned
func nullUUID(id uuid.UUID) null.String {
	if id.String() == uuid.Nil.String() {
		return null.String{}
	}

	return null.StringFrom(id.String())
}
