package serverservice

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"go.hollow.sh/serverservice/internal/models"
)

var (
	errComponentFirmwareSetPayload = errors.New("error in component firmware set payload")
)

func (r *Router) serverComponentFirmwareSetList(c *gin.Context) {
	pager := parsePagination(c)

	var params ComponentFirmwareSetListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		badRequestResponse(c, "invalid filter payload: ComponentFirmwareSetListParams{}", err)
		return
	}

	mods := params.queryMods()

	count, err := models.ComponentFirmwareSets(mods...).Count(c.Request.Context(), r.DB)
	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	// add pagination
	pager.Preload = false
	pager.OrderBy = models.ComponentFirmwareSetColumns.CreatedAt + " DESC"

	mods = append(mods, pager.serverQueryMods()...)

	dbFirmwareSets, err := models.ComponentFirmwareSets(mods...).All(c.Request.Context(), r.DB)
	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	firmwareSets := make([]ComponentFirmwareSet, 0, count)

	for _, dbFS := range dbFirmwareSets {
		f := ComponentFirmwareSet{}
		if err := f.fromDBModel(dbFS); err != nil {
			failedConvertingToVersioned(c, err)
			return
		}

		firmwareSets = append(firmwareSets, f)
	}

	pd := paginationData{
		pageCount:  len(firmwareSets),
		totalCount: count,
		pager:      pager,
	}

	listResponse(c, firmwareSets, pd)
}

func (r *Router) serverComponentFirmwareSetGet(c *gin.Context) {
	mods := []qm.QueryMod{
		qm.Where("id=?", c.Param("uuid")),
	}

	dbFirmwareSet, err := models.ComponentFirmwareSets(mods...).One(c.Request.Context(), r.DB)
	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	var firmwareSet ComponentFirmwareSet
	if err = firmwareSet.fromDBModel(dbFirmwareSet); err != nil {
		failedConvertingToVersioned(c, err)
		return
	}

	itemResponse(c, firmwareSet)
}

func (r *Router) serverComponentFirmwareSetCreate(c *gin.Context) {
	var firmwareSetPayload ComponentFirmwareSetPayload

	if err := c.ShouldBindJSON(&firmwareSetPayload); err != nil {
		badRequestResponse(c, "invalid payload: ComponentFirmwareSetCreate{}", err)
		return
	}

	// validate and collect firmware UUIDs
	firmwareUUIDs := []uuid.UUID{}

	if len(firmwareSetPayload.ComponentFirmwareUUIDs) == 0 {
		badRequestResponse(
			c,
			"",
			errors.Wrap(errComponentFirmwareSetPayload, "expected one or more firmware UUIDs, got none"),
		)

		return
	}

	for _, firmwareUUID := range firmwareSetPayload.ComponentFirmwareUUIDs {
		// parse uuid
		firmwareUUIDParsed, err := uuid.Parse(firmwareUUID)
		if err != nil {
			badRequestResponse(
				c,
				"invalid firmware UUID: "+firmwareUUID,
				errors.Wrap(errComponentFirmwareSetPayload, err.Error()),
			)

			return
		}

		// validate component firmware version exists
		exists, err := models.ComponentFirmwareVersionExists(c.Request.Context(), r.DB, firmwareUUID)
		if err != nil {
			dbErrorResponse(c, err)

			return
		}

		if !exists {
			badRequestResponse(
				c,
				"",
				errors.Wrap(errComponentFirmwareSetPayload, "firmware UUID does not exist: "+firmwareUUID),
			)

			return
		}

		firmwareUUIDs = append(firmwareUUIDs, firmwareUUIDParsed)
	}

	dbFirmwareSet, err := firmwareSetPayload.toDBModelFirmwareSet()
	if err != nil {
		badRequestResponse(c, "invalid db model: ComponentFirmwareSet", err)
		return
	}

	// being transaction to insert a new firmware set and its mapping
	tx, err := r.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	// nolint:errcheck // TODO(joel): log error
	defer tx.Rollback()

	// insert set
	if err := dbFirmwareSet.Insert(c.Request.Context(), tx, boil.Infer()); err != nil {
		dbErrorResponse(c, err)
		return
	}

	// insert set - firmware mapping
	for _, id := range firmwareUUIDs {
		m := models.ComponentFirmwareSetMap{FirmwareSetID: dbFirmwareSet.ID, FirmwareID: id.String()}

		err := m.Insert(c.Request.Context(), tx, boil.Infer())
		if err != nil {
			dbErrorResponse(c, err)
			return
		}
	}

	// commit
	if err := tx.Commit(); err != nil {
		dbErrorResponse(c, err)
		return
	}

	createdResponse(c, dbFirmwareSet.ID)
}

func (r *Router) serverComponentFirmwareSetDelete(c *gin.Context) {
	dbFirmware, err := r.loadComponentFirmwareSetFromParams(c)
	if err != nil {
		return
	}

	if _, err = dbFirmware.Delete(c.Request.Context(), r.DB); err != nil {
		dbErrorResponse(c, err)
		return
	}

	deletedResponse(c)
}

func (r *Router) serverComponentFirmwareSetUpdate(c *gin.Context) {
	dbFirmware, err := r.loadComponentFirmwareSetFromParams(c)
	if err != nil {
		return
	}

	var newValues ComponentFirmwareSet
	if err := c.ShouldBindJSON(&newValues); err != nil {
		badRequestResponse(c, "invalid payload: ComponentFirmwareSet{}", err)
		return
	}

	dbFirmware.Name = newValues.Name

	// TODO (joel): update firmware ids in set

	cols := boil.Infer()

	if _, err := dbFirmware.Update(c.Request.Context(), r.DB, cols); err != nil {
		dbErrorResponse(c, err)
		return
	}

	updatedResponse(c, dbFirmware.ID)
}

func (r *Router) loadComponentFirmwareSetFromParams(c *gin.Context) (*models.ComponentFirmwareSet, error) {
	u, err := r.parseUUID(c)
	if err != nil {
		return nil, err
	}

	firmwareSet, err := models.FindComponentFirmwareSet(c.Request.Context(), r.DB, u.String())
	if err != nil {
		dbErrorResponse(c, err)

		return nil, err
	}

	return firmwareSet, nil
}
