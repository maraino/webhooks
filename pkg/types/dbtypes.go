package types

import (
	"context"
	"database/sql"
)

// Device represents a row of the devices table.
type Device struct {
	ID        string
	Type      string
	Owner     string
	Allow     bool
	Data      []byte
	CreatedAt sql.NullInt64
}

// LoadDevice returns a device from the dtabase by id.
func LoadDevice(ctx context.Context, db *sql.DB, id string) (*Device, error) {
	const selectQuery = "SELECT id, type, owner, allow, data, created_at FROM devices WHERE id = :id"

	var device Device
	row := db.QueryRowContext(ctx, selectQuery, sql.Named("id", id))
	if err := device.Scan(row); err != nil {
		return nil, err
	}
	return &device, nil
}

// Scan scans the given now into the device struct.
func (s *Device) Scan(row *sql.Row) error {
	return row.Scan(&s.ID, &s.Type, &s.Owner, &s.Allow, &s.Data, &s.CreatedAt)
}
