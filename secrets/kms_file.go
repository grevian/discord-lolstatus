package secrets

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
)

type KMSFileSecretProvider struct {
	secrets interface{}
}

func CreateKeyPath(projectID string, locationID string, keyRingID string, cryptoKeyID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		projectID, locationID, keyRingID, cryptoKeyID)
}

func NewKMSFileSecretProvider(filePath string, keyPath string, secrets interface{}) (SecretProvider, error) {
	// Read the encrypted secrets from disk
	encDat, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Authorize using the application default credentials
	ctx := context.Background()
	client, err := google.DefaultClient(ctx, cloudkms.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	// Instantiate a KMS client
	kmsService, err := cloudkms.New(client)
	if err != nil {
		return nil, err
	}

	// Ask for our encrypted secrets to be decrypted
	req := &cloudkms.DecryptRequest{
		Ciphertext: base64.StdEncoding.EncodeToString(encDat),
	}
	resp, err := kmsService.Projects.Locations.KeyRings.CryptoKeys.Decrypt(keyPath, req).Do()
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(resp.Plaintext)
	if err != nil {
		return nil, err
	}

	// Read our unencrypted secrets to the requested format
	err = json.Unmarshal(decoded, secrets)
	if err != nil {
		return nil, err
	}

	return &KMSFileSecretProvider{
		secrets: secrets,
	}, nil
}

func (r *KMSFileSecretProvider) GetSecrets() interface{} {
	return r.secrets
}
