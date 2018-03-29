package secrets

import (
	"encoding/json"
	"io/ioutil"
)

type RawFileSecretProvider struct {
	secrets interface{}
}

func NewRawFileSecretProvider(filePath string, secrets interface{}) (SecretProvider, error) {
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(dat, secrets)
	if err != nil {
		return nil, err
	}

	return &RawFileSecretProvider{
		secrets: secrets,
	}, nil
}

func (r *RawFileSecretProvider) GetSecrets() interface{} {
	return r.secrets
}
