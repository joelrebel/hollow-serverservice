package serverservice

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"go.hollow.sh/serverservice/internal/models"
)

// ComponentFirmwareSet represents a group of firmwares
type ComponentFirmwareSet struct {
	UUID              uuid.UUID                  `json:"uuid"`
	Name              string                     `json:"name"`
	Metadata          json.RawMessage            `json:"metadata"`
	ComponentFirmware []ComponentFirmwareVersion `json:"component_firmware"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
}

func (s *ComponentFirmwareSet) fromDBModel(dbFS *models.ComponentFirmwareSet) error {
	var err error

	s.UUID, err = uuid.Parse(dbFS.ID)
	if err != nil {
		return err
	}

	s.Name = dbFS.Name
	s.CreatedAt = dbFS.CreatedAt.Time
	s.UpdatedAt = dbFS.UpdatedAt.Time

	return nil
}

// ComponentFirmwareSetPayload represents the payload to create a firmware set
type ComponentFirmwareSetPayload struct {
	ID                     uuid.UUID       `json:"uuid"`
	Name                   string          `json:"name" binding:"required"`
	Metadata               json.RawMessage `json:"metadata"`
	ComponentFirmwareUUIDs []string        `json:"component_firmware_uuids"`
}

func (sc *ComponentFirmwareSetPayload) toDBModelFirmwareSet() (*models.ComponentFirmwareSet, error) {
	return &models.ComponentFirmwareSet{
		ID:   uuid.NewString(),
		Name: sc.Name,
	}, nil
}
