package model

// CSLProfile is the response for GET /{username}.json
type CSLProfile struct {
	Username string            `json:"username"`
	Textures map[string]string `json:"textures,omitempty"`
	Skin     string            `json:"skin,omitempty"`
	Cape     string            `json:"cape,omitempty"`
	Elytra   string            `json:"elytra,omitempty"`
}
