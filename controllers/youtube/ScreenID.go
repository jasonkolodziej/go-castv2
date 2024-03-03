package youtube

import "github.com/jasonkolodziej/go-castv2/primitives"

// GetScreenIDRequest for getting a screen ID for an existing youtube application.
type GetScreenIDRequest struct {
	primitives.PayloadHeaders
	ScreenID int `json:"screen_ids"`
}
