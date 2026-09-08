package epcadopt

import "context"

// DeviceStatus is what the controller reports about a specific device's
// checkin/adoption state. The exact response body shape is still unconfirmed
// (no concrete Client implementation exists yet — this struct is this
// package's own intended shape, to be filled in from whatever the real
// per-device status endpoint returns, e.g.
// `GET /api/v1/orgs/{org_id}/hvs/{hv_id}/networks/{network_id}/devices/aps/{device_id}`).
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
// T0.1/T0.2 are now resolved against a real controller's own OpenAPI schema
// and empirical auth probing (2026-09-07): a stable REST API exists under
// `/api/v1/`, scoped org -> hv (hierarchy view) -> network -> device (three
// levels, not two — see Request.HVID). Two SEPARATE auth models exist:
//   - User-facing endpoints (Login, RegisterDevice, DeviceStatus below) sit
//     behind a custom JWT-bearer check; confirmed empirically by an
//     unauthenticated call returning a clean `401 {"code":401,"message":
//     "Not authenticated"}` rather than a crash or a different shape.
//   - The device's OWN checkin (a *device*-to-controller call, not something
//     this Client needs to make — the AP does it) uses a completely
//     different, header-based HMAC scheme, NOT this Client's auth. Do not
//     reuse Client's Login for anything checkin-related.
//
// No concrete implementation exists yet (Phase 2). Registration is
// confirmed to go through the real API (no direct datastore access needed)
// — likely POST `/api/v1/orgs/{org_id}/hvs/{hv_id}/networks/{network_id}/devices`,
// but the exact request/response body shape still needs confirming before
// implementing RegisterDevice/DeviceStatus for real.
type Client interface {
	// Login authenticates to the controller's own management surface via
	// its JWT-bearer scheme (see above). Exact request shape (credentials
	// body vs. a separate token endpoint) still needs confirming.
	Login(ctx context.Context, user, pass string) error

	// RegisterDevice adds a device to controller inventory under the given
	// org/hv/network scope (Request.OrgID/HVID/NetworkID). Confirmed to be a
	// real API call, not datastore access — exact body shape unconfirmed.
	RegisterDevice(ctx context.Context, req Request) error

	// DeviceStatus reads back a specific device's checkin/adoption state,
	// keyed by its real serial (never a generated identity).
	DeviceStatus(ctx context.Context, realSerial string) (DeviceStatus, error)
}
