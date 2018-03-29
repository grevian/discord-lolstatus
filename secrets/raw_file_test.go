package secrets

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFile(t *testing.T) {
	// Create a test secrets interface and some content
	type testSecrets struct {
		TestKey string
		TestMap map[string]string
	}

	// An example of secret file content
	secretString := `{ "TestKey": "FooBar", "TestMap": { "FooKeyA": "A", "FooKeyB": "B" } }`

	// Create a tempfile to test loading secrets from
	f, err := ioutil.TempFile("./", "secrets-test-")
	require.NoError(t, err)
	// And make sure it's cleaned up once we're done
	defer func() { assert.NoError(t, os.Remove("./"+f.Name())) }()

	// Write our sample secrets out to the test file
	n, err := f.WriteString(secretString)
	f.Close()
	require.Equal(t, len(secretString), n)

	// Attempt to load secrets from the test file
	secretVar := &testSecrets{}
	var secrets SecretProvider
	secrets, err = NewRawFileSecretProvider(f.Name(), secretVar)

	// Ensure at least something was loaded
	require.NoError(t, err)
	require.NotNil(t, secrets)

	// Test that the correct thing was loaded
	s := secrets.GetSecrets().(*testSecrets)
	assert.Equal(t, "FooBar", s.TestKey)
	assert.Equal(t, "A", s.TestMap["FooKeyA"])
	assert.Equal(t, "B", s.TestMap["FooKeyB"])
}
