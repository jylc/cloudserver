package models

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/jylc/cloudserver/pkg/hashid"
	"net/url"
)

func (user User) WebAuthnID() []byte {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, uint64(user.ID))
	return bs
}

func (user User) WebAuthnName() string {
	return user.Email
}

func (user User) WebAuthnDisplayName() string {
	return user.Nick
}

func (user User) WebAuthnIcon() string {
	avatar, _ := url.Parse("/api/v3/user/avatar/" + hashid.HashID(user.ID, hashid.UserID))
	base := GetSiteURL()
	base.Scheme = "https"
	return base.ResolveReference(avatar).String()
}

func (user User) WebAuthnCredentials() []webauthn.Credential {
	var res []webauthn.Credential
	err := json.Unmarshal([]byte(user.Authn), &res)
	if err != nil {
		fmt.Println(err)
	}
	return res
}

func (user *User) RegisterAuthn(credential *webauthn.Credential) error {
	exists := user.WebAuthnCredentials()
	exists = append(exists, *credential)
	res, err := json.Marshal(exists)
	if err != nil {
		return err
	}
	return Db.Model(user).Update("authn", string(res)).Error
}

func (user *User) RemoveAuthn(id string) {
	exists := user.WebAuthnCredentials()
	for i := 0; i < len(exists); i++ {
		idEncoding := base64.StdEncoding.EncodeToString(exists[i].ID)
		if idEncoding == id {
			exists[len(exists)-1], exists[i] = exists[i], exists[len(exists)-1]
			exists = exists[:len(exists)-1]
			break
		}
	}
	res, _ := json.Marshal(exists)
	Db.Model(user).Update("authn", string(res))
}
