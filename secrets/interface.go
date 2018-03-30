package secrets

type SecretProvider interface {
	GetSecrets() interface{}
}
