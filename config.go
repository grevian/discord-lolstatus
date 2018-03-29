package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	riotapi "github.com/yuhanfang/riot/apiclient"
	"github.com/yuhanfang/riot/ratelimit"

	"./secrets"
)

const (
	RAW_FILE_PROVIDER = "RawFile"
	KMS_FILE_PROVIDER = "KMSFile"
)

type botConfiguration struct {
	// Tokens for our secret handling
	credentials *botSecrets
	secretProvider
}

type secretProvider struct {
	ProviderType       string
	ProviderLocation   string
	ProviderAdditional string
}

type botSecrets struct {
	RiotApiKey       string
	DiscordAuthToken string
}

func NewBotConfiguration(filePath string) (*botConfiguration, error) {
	// Load the basic configuration from the given file
	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	configuration := &botConfiguration{
		credentials: &botSecrets{},
	}
	err = json.Unmarshal(fileContents, configuration)
	if err != nil {
		return nil, err
	}

	// Load encrypted secrets using the specified provider
	var provider secrets.SecretProvider
	switch configuration.secretProvider.ProviderType {
	case RAW_FILE_PROVIDER:
		log.WithField("ProviderLocation", configuration.secretProvider.ProviderLocation).Debug("Loading secrets from raw file")
		provider, err = secrets.NewRawFileSecretProvider(configuration.secretProvider.ProviderLocation, &botSecrets{})
	case KMS_FILE_PROVIDER:
		log.WithFields(log.Fields{
			"ProviderLocation":   configuration.secretProvider.ProviderLocation,
			"ProviderAdditional": configuration.secretProvider.ProviderAdditional,
		}).Debug("Loading secrets using file encrypted with KMS")
		KeyPath := configuration.secretProvider.ProviderAdditional
		provider, err = secrets.NewKMSFileSecretProvider(configuration.secretProvider.ProviderLocation, KeyPath, configuration.credentials)
	default:
		err = fmt.Errorf("unknown secrets provider: %s", configuration.secretProvider.ProviderType)
	}

	if err != nil {
		return nil, err
	}

	// Load the decrypted secrets into our configuration
	configuration.credentials = provider.GetSecrets().(*botSecrets)

	return configuration, nil
}

func (b *botConfiguration) GetRiotClient() (riotapi.Client, error) {
	key := b.credentials.RiotApiKey
	if key == "" {
		return nil, fmt.Errorf("no RIOT API Key")
	}

	httpClient := http.DefaultClient
	limiter := ratelimit.NewLimiter()
	client := riotapi.New(key, httpClient, limiter)
	return client, nil
}

func (b *botConfiguration) GetDiscordClient() (*discordgo.Session, error) {
	discordAuthToken := b.credentials.DiscordAuthToken
	if discordAuthToken == "" {
		return nil, fmt.Errorf("no Discord Auth Token")
	}

	discord, err := discordgo.New("Bot " + discordAuthToken)
	if err != nil {
		return nil, err
	}

	err = discord.Open()
	if err != nil {
		return nil, err
	}

	return discord, nil
}
