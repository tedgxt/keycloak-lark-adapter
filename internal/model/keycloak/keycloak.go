package keycloak

import "fmt"

// TokenResp defines token respond from keycloak
type TokenResp struct {
	AccessToken string `json:"access_token"`
}

// User defines user profile info
type User struct {
	Id            string                 `json:"id,omitempty"`
	Enabled       *bool                  `json:"enabled,omitempty"`
	Attributes    map[string]interface{} `json:"attributes"`
	Username      string                 `json:"username,omitempty"`
	EmailVerified bool                   `json:"emailVerified"`
	Email         string                 `json:"email,omitempty"`
	FirstName     string                 `json:"firstName"`
	LastName      string                 `json:"lastName"`
}

func (u *User) String() string {
	return fmt.Sprintf("{Id: %v, Username: %v, Enabled: %v, Attributes: %#v, Email: %v, EmailVerified: %v, FirstName: %v, LastName: %v}",
		u.Id, u.Username, *u.Enabled, u.Attributes, u.Email, u.EmailVerified, u.FirstName, u.LastName)
}

type GroupAssignment struct {
	GroupID string `json:"groupId"`
	Realm   string `json:"realm"`
	UserID  string `json:"userId"`
}

type GroupInfo struct {
	Access      map[string]interface{} `json:"access,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	ClientRoles map[string]interface{} `json:"clientRoles,omitempty"`
	RealmRoles  []string               `json:"realmRoles,omitempty"`
	Path        string                 `json:"path,omitempty"`
	Name        string                 `json:"name,omitempty"`
	SubGroups   []*GroupInfo           `json:"subGroups,omitempty"`
	ID          string                 `json:"id,omitempty"`
}
