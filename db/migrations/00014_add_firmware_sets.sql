-- +goose Up
-- +goose StatementBegin

CREATE TABLE component_firmware_set (
  id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  name STRING NOT NULL,
  metadata JSONB NULL,
  created_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NULL,
  UNIQUE INDEX idx_name (name)
);

-- maps a component_firmware_set ID to one or more component firmware version ID(s)
CREATE TABLE component_firmware_set_map (
  id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  firmware_set_id UUID NOT NULL REFERENCES component_firmware_set(id) ON DELETE CASCADE,
  firmware_id UUID NOT NULL REFERENCES component_firmware_version(id) ON DELETE CASCADE
);

ALTER TABLE servers ADD COLUMN firmware_set_id UUID REFERENCES component_firmware_set(id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

SET sql_safe_updates = false;
DROP TABLE component_firmware_sets;

DROP TABLE component_firmware_set_map;

ALTER TABLE servers DROP COLUMN firmware_set_id;

-- +goose StatementEnd
