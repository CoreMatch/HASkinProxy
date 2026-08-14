package model

// ProfileRequest is used for POST /api/profiles/minecraft
type ProfileRequest []string

// ProfileResponse is the item in the response array of POST /api/profiles/minecraft
type ProfileResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SessionProfileResponse is the response of GET /sessionserver/session/minecraft/profile/:uuid
type SessionProfileResponse struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Properties []ProfileProperty `json:"properties"`
}

type ProfileProperty struct {
	Name      string `json:"name"`
	Value     string `json:"value"` // base64 encoded JSON
	Signature string `json:"signature,omitempty"`
}

// TexturePropertyValue is the decoded JSON from ProfileProperty.Value
type TexturePropertyValue struct {
	Timestamp uint64 `json:"timestamp"`
	ProfileID string `json:"profileId"`
	ProfileName string `json:"profileName"`
	Textures struct {
		SKIN *TextureInfo `json:"SKIN,omitempty"`
		CAPE *TextureInfo `json:"CAPE,omitempty"`
		ELYTRA *TextureInfo `json:"ELYTRA,omitempty"`
	} `json:"textures"`
}

type TextureInfo struct {
	URL      string            `json:"url"`
	Metadata map[string]string `json:"metadata,omitempty"` // e.g., model: slim
}
