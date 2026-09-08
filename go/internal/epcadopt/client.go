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
// T0.1/T0.2 are resolved against a real controller's own OpenAPI schema and
// empirical auth probing (2026-09-07): a stable REST API exists under
// `/api/v1/`, scoped org -> hv (hierarchy view) -> network -> device (three
// levels, not two — see Request.HVID). Two SEPARATE auth models exist:
//   - User-facing endpoints (RegisterDevice, DeviceStatus below) sit behind a
//     custom JWT-bearer check; confirmed empirically by an unauthenticated
//     call returning a clean `401 {"code":401,"message":"Not authenticated"}`
//     rather than a crash or a different shape.
//   - The device's OWN checkin (a *device*-to-controller call, not something
//     this Client needs to make — the AP does it) uses a completely
//     different, header-based HMAC scheme, NOT this Client's auth. Do not
//     reuse Client's Login for anything checkin-related.
//
// Login itself is a SEPARATE, still-open question (re-probed 2026-09-07,
// partially re-opening T0.1): the one route that looked like an obvious
// login endpoint, `POST /api/v1/jwt-token`, is itself gated behind the SAME
// JWT-bearer check as every other user-facing route — confirmed by
// distinguishing two different 401 bodies from that same endpoint: sending no
// Authorization header returns `{"code":401,"message":"Not authenticated"}`,
// while sending a syntactically-plausible-but-invalid bearer token returns a
// DIFFERENT body, `{"code":401,"message":"Could not validate credentials"}`
// (i.e. it actually tries to decode/verify whatever token is presented before
// touching the request body at all). A route that requires an existing valid
// token to reach its own body-validation logic cannot be the initial
// human-login entry point — so wherever a real operator/UI actually obtains
// their first token is still unconfirmed. It is likely one of: a route this
// probe didn't enumerate, a session established by a different component in
// the stack (the controller is a multi-container stack; only the API
// container's own route table was inspected), or an external identity
// provider the SPA talks to before ever calling this API. Do NOT implement
// Login() against the naive "POST grant_type/username/password to
// /api/v1/jwt-token and expect a token back" assumption — that call is
// confirmed to 401 unconditionally without a pre-existing token, so it cannot
// work as a first login step no matter what body is sent.
//
// No concrete implementation exists yet (Phase 2), and Login specifically
// cannot be implemented until the real first-login path is found. Device
// registration is confirmed to go through the real API (no direct datastore
// access needed) — likely POST
// `/api/v1/orgs/{org_id}/hvs/{hv_id}/networks/{network_id}/devices`, but the
// exact request/response body shape still needs confirming before
// implementing RegisterDevice/DeviceStatus for real.
type Client interface {
	// Login authenticates to the controller's own management surface via
	// its JWT-bearer scheme (see above). NOT YET IMPLEMENTABLE: the obvious
	// candidate endpoint (`POST /api/v1/jwt-token`) is confirmed to require a
	// pre-existing valid token itself, so it cannot be the first-login step —
	// see the doc comment above. The real first-login mechanism is still
	// unconfirmed.
	Login(ctx context.Context, user, pass string) error

	// RegisterDevice adds a device to controller inventory under the given
	// org/hv/network scope (Request.OrgID/HVID/NetworkID). Confirmed to be a
	// real API call, not datastore access — exact body shape unconfirmed.
	RegisterDevice(ctx context.Context, req Request) error

	// DeviceStatus reads back a specific device's checkin/adoption state,
	// keyed by its real serial (never a generated identity).
	DeviceStatus(ctx context.Context, realSerial string) (DeviceStatus, error)
}
