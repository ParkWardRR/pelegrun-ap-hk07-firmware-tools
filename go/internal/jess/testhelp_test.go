package jess

import (
	"encoding/json"
	"net/http"
)

func json_decode(r *http.Request, v any) { _ = json.NewDecoder(r.Body).Decode(v) }
