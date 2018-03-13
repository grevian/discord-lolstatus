package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	riotapi "github.com/yuhanfang/riot/apiclient"
	"github.com/yuhanfang/riot/constants/region"

	"context"
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"
)

const BOT_STATE = "./botdata.json"

// A simplified struct with public fields, used to write the summoner details to disk
type SerializedSummonerDetails struct {
	LolSummonerId    int64
	DiscordChannelId string
	LastGameReported int64
}

type botStorage struct {
	MonitoredSummoners map[string]*SummonerDetails
	sync.RWMutex
}

func (s *SummonerDetails) flatten() *SerializedSummonerDetails {
	s.Lock()
	defer s.Unlock()

	flattened := &SerializedSummonerDetails{
		LolSummonerId:    s.summoner.ID,
		DiscordChannelId: s.reportingChannel.ID,
		LastGameReported: s.lastGameReported,
	}

	return flattened
}

func (b *botStorage) store() ([]byte, error) {
	// Simplify internal models so they can be written to disk
	flattened := make(map[string]*SerializedSummonerDetails)

	for k, s := range b.MonitoredSummoners {
		flattened[k] = s.flatten()
		log.WithField("summoner", k).Debug("Writing Summoner to flattened index")
	}

	bytes, err := json.Marshal(flattened)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (b *botStorage) load(bytes []byte, client riotapi.Client, session *discordgo.Session) error {
	var flattened = make(map[string]SerializedSummonerDetails)

	if err := json.Unmarshal(bytes, &flattened); err != nil {
		return err
	}

	for k, v := range flattened {
		ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
		summoner, err := client.GetBySummonerID(ctx, region.NA1, v.LolSummonerId)
		if err != nil {
			return err
		}

		channel, err := session.Channel(v.DiscordChannelId)
		if err != nil {
			return err
		}

		s := SummonerDetails{
			summoner:         summoner,
			reportingChannel: channel,
			lastGameReported: v.LastGameReported,
		}
		b.MonitoredSummoners[k] = &s
	}

	return nil
}

// Save the bots state to disk
func (bot *LeagueAnnouncerBot) persist() error {
	log.Debug("Writing bot state to disk")
	// Write the current state of the bot to disk, to load for next time
	storageBytes, err := bot.storage.store()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(BOT_STATE, storageBytes, 0600)

	if err != nil {
		return err
	}
	return nil
}

// Try to load the bots state from disk, if it exists
func (bot *LeagueAnnouncerBot) loadState() error {
	// Look for a state file and read it in
	bytes, err := ioutil.ReadFile(BOT_STATE)
	if err != nil {
		return err
	}

	// Load the state into memory, loading summoner details and discord channels
	botState := botStorage{MonitoredSummoners: make(map[string]*SummonerDetails)}
	err = botState.load(bytes, bot.riot, bot.discord)
	if err != nil {
		return err
	}

	// Overwrite the bots current state with what we loaded from disk
	bot.storage = botState
	log.Debugf("Loaded bot state from disk, Loaded %d users", len(botState.MonitoredSummoners))
	return nil
}
