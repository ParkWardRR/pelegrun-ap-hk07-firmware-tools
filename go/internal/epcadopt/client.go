package epcadopt

import "context"

// DeviceStatus is what the controller reports about a specific device's
// checkin/adoption state. Field names are a best-guess shape pending T0.1/T0.2
// (specs/002-epc-pairing/tasks.md Phase 0) — expect this to change once a real
// controller's actual response shape is confirmed.
type DeviceStatus struct {
	Registered  bool
	Adopted     bool
	LastCheckin string
}

// Client is the narrow interface epcadopt needs from an EPC controller.
// Deliberately kept to three methods until a real controller confirms the
// actual mechanism (see specs/002-epc-pairing/plan.md's package-shape
// section) — resist adding more before then.
//
// No concrete implementation exists yet. If T0.2 finds that reliable
// registration requires direct datastore access rather than a stable API,
// model that as a separate DatastoreSeeder interface instead of folding it
// in here (see plan.md) so the higher-risk path stays clearly labeled.
type Client interface {
	// Login authenticates to the controller's own management surface.
	// Auth model (cookie/bearer/CSRF) is unconfirmed — see T0.1.
	Login(ctx context.Context, user, pass string) error

	// RegisterDevice adds a device to controller inventory under the given
	// org/network scope. Whether this is a stable API call or requires
	// datastore access is unconfirmed — see T0.2.
	RegisterDevice(ctx context.Context, req Request) error

	// DeviceStatus reads back a specific device's checkin/adoption state,
	// keyed by its real serial (never a generated identity).
	DeviceStatus(ctx context.Context, realSerial string) (DeviceStatus, error)
}
