package serverservice

import (
	"net/url"

	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"go.hollow.sh/serverservice/internal/models"
)

// ComponentFirmwareSetListParams allows you to filter the results
type ComponentFirmwareSetListParams struct {
	Name                string `form:"name"`
	Pagination          *PaginationParams
	AttributeListParams []AttributeListParams
}

func (p *ComponentFirmwareSetListParams) setQuery(q url.Values) {
	if p == nil {
		return
	}

	if p.Name != "" {
		q.Set("name", p.Name)
	}

	encodeAttributesListParams(p.AttributeListParams, "attr", q)

	p.Pagination.setQuery(q)
}

// queryMods converts the list params into sql conditions that can be added to sql queries
func (p *ComponentFirmwareSetListParams) queryMods() []qm.QueryMod {
	mods := []qm.QueryMod{}

	if p.Name != "" {
		m := models.ComponentFirmwareSetWhere.Name.EQ(p.Name)
		mods = append(mods, m)
	}

	return mods
}
