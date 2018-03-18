package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	riotapi "github.com/yuhanfang/riot/apiclient"
	"github.com/yuhanfang/riot/ratelimit"
	"github.com/yuhanfang/riot/constants/region"

	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var botMain *LeagueAnnouncerBot

type SummonerDetails struct {
	summoner         *riotapi.Summoner
	reportingChannel *discordgo.Channel
	lastGameReported int64
	sync.RWMutex
}

type LeagueAnnouncerBot struct {
	discord *discordgo.Session
	riot    riotapi.Client
	storage botStorage
}

func main() {
	discord, err := setupDiscord()
	if err != nil {
		log.WithError(err).Panic("Failed to connect discord client")
	}

	riot, err := setupRiot()
	if err != nil {
		log.WithError(err).Panic("Failed to connect riot API client")
	}

	bot := &LeagueAnnouncerBot{
		discord: discord,
		riot:    riot,
		storage: botStorage{
			MonitoredSummoners: make(map[string]*SummonerDetails),
		},
	}

	// Try to load the bots state from disk, if it exists
	err = bot.loadState()
	if err != nil {
		log.WithError(err).Warn("Could not load bot state from disk, proceeding with empty state")
	}

	botMain = bot

	// Start monitoring any summoners who were loaded from disk
	for _, s := range botMain.storage.MonitoredSummoners {
		log.Debugf("Monitoring %s", s.summoner.Name)
		go monitorLoop(s)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Write the current state of the bot to disk, to load for next time
	err = botMain.persist()
	if err != nil {
		log.WithError(err).Error("Could not serialize bot state")
	}

	// Cleanly close down the Discord session.
	botMain.discord.Close()
}

func setupRiot() (riotapi.Client, error) {
	key := os.Getenv("RIOT_APIKEY")
	if key == "" {
		return nil, fmt.Errorf("no RIOT API Key")
	}

	httpClient := http.DefaultClient
	limiter := ratelimit.NewLimiter()
	client := riotapi.New(key, httpClient, limiter)
	return client, nil
}

func setupDiscord() (*discordgo.Session, error) {
	discordAuthToken := os.Getenv("DISCORD_AUTH")
	if discordAuthToken == "" {
		return nil, fmt.Errorf("no Discord Auth Token Provided")
	}

	discord, err := discordgo.New("Bot " + discordAuthToken)
	if err != nil {
		return nil, err
	}

	discord.AddHandler(messageHandler)

	err = discord.Open()
	if err != nil {
		return nil, err
	}

	return discord, nil
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore our own messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		log.WithError(err).Error("Failed to get channel to report/respond in")
		return
	}

	if strings.HasPrefix(m.Content, "!skill") {
		//TODO: Finish this command
		//cmdSkill(m.Content, channel, s, m)
	}

	if strings.HasPrefix(m.Content, "!leaguewatch") {
		cmdLeagueWatch(m.Content, channel, s, m)
	}

	if strings.HasPrefix(m.Content, "!help") {
		cmdHelp(s, m)
	}


}

func cmdSkill(champ string, channel *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) {
	//TODO: Finish this command
	// Check if it's a command we recognize

		tokens := strings.Split(champ, " ")
		if len(tokens) != 3 {
			// Unrecognized, must be in the form: "!skill champ q
			log.Debug("Ignoring a unrecognized command: %s", m.Content)
			s.ChannelMessageSend(m.ChannelID, "Unrecognized command, try '!skill Karma q'")
			return
		}
		//championName := tokens[1]
		//championSkill := tokens[2]
		//ctx := context.Background()
		//champList, err := botMain.riot.GetChampions(ctx, region.NA1)

		//for _, v := range champList.Champions {
		//	if v.Name == championName {
		//		//championID := something
		//	}
		//}

		//ctx,_  := context.WithTimeout(context.Background(), 5*time.Second)
		//c := riotapi.staticdata.New(http.DefaultClient)
		//versions,_ err := c.Versions(ctx)
		//champs,_  := c.Champions(ctx, versions[0], language.EnglishUnitedStates)

		//for WhatIsThis, champ := range champs.Data {
		//	for _, ability := range champ.Spells {
		//		fmt.Print(ability.Description)
		//	}
		//}


		//if err != nil {
		//	// Handle this error, print to the users
		//	log.WithError(err).Error("Failed to get find details")
		//	s.ChannelMessageSend(m.ChannelID, "Could not find that champion: "+err.Error())
		//	return
		//}
}


func cmdLeagueWatch(content string, channel *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) {

	// Check if it's a command we recognize

		tokens := strings.Split(content, " ")
		if len(tokens) != 2 {
			// Unrecognized, must be in the form: "!leaguewatch SomeGuy
			log.Debug("Ignoring a unrecognized command: %s", m.Content)
			s.ChannelMessageSend(m.ChannelID, "Unrecognized command, try '!leaguewatch <summonername>'")
			return
		}
		watchTarget := tokens[1]
		ctx := context.Background()
		summoner, err := botMain.riot.GetBySummonerName(ctx, region.NA1, watchTarget)
		if err != nil {
			// Handle this error, print to the users
			log.WithError(err).Error("Failed to get find details")
			s.ChannelMessageSend(m.ChannelID, "Could not find that summoner: "+err.Error())
			return
		}

		go startMonitoring(summoner, channel)


}

func cmdHelp(s *discordgo.Session, m *discordgo.MessageCreate) {

	helpMessage := `discord-lolstatus by Grevian, bastardized by foghsho
Commands:
!leaguewatch <summonername> - This will monitor an account to show how bad they are after each game
!help - That's this! idiot.`

	s.ChannelMessageSend(m.ChannelID, helpMessage)

}

func startMonitoring(summoner *riotapi.Summoner, reportingChannel *discordgo.Channel) {
	if _, exists := botMain.storage.MonitoredSummoners[summoner.Name]; exists {
		// If we're already monitoring this person, bail out immediately
		log.Infof("Was asked to monitor %s, but we're already monitoring them", summoner.Name)
		botMain.discord.ChannelMessageSend(reportingChannel.ID, "I'm already watching "+summoner.Name)
		return
	}

	sd := &SummonerDetails{
		summoner:         summoner,
		lastGameReported: 0,
		reportingChannel: reportingChannel,
	}

	botMain.storage.Lock()
	botMain.storage.MonitoredSummoners[summoner.Name] = sd
	botMain.storage.Unlock()

	monitorLoop(sd)
}

func monitorLoop(sd *SummonerDetails) {
	for {
		time.Sleep(10 * time.Second)

		// Get the last played matches for this player
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		matchList, err := botMain.riot.GetRecentMatchlist(ctx, region.NA1, sd.summoner.AccountID)
		if err != nil {
			continue
		}

		// See if their most recent match has changed since we last reported
		currentGame := matchList.Matches[0].GameID

		// If there's a new game available, get the match details, then generate and post a report for it
		if sd.lastGameReported != currentGame {
			ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
			match, err := botMain.riot.GetMatch(ctx, region.NA1, currentGame)
			if err != nil {
				log.WithError(err).WithField("match", currentGame).Warn("Failed to get match details")
				continue
			}

			// Scan the match participantIdentities for our summoner
			var gamePlayer riotapi.ParticipantIdentity
			for _, p := range match.ParticipantIdentities {
				if p.Player.AccountID == sd.summoner.AccountID {
					gamePlayer = p
				}
			}

			// Pull the summoners player data from this match
			var gameParticipant riotapi.Participant
			for _, p := range match.Participants {
				if gamePlayer.ParticipantID != p.ParticipantID {
					continue
				}
				gameParticipant = p
			}

			// Build the message and send it to the channel
			statline := fmt.Sprintf("%d/%d/%d",
				gameParticipant.Stats.Kills,
				gameParticipant.Stats.Deaths,
				gameParticipant.Stats.Assists)

			// Build the champion message and send to channel
			var champid = gameParticipant.ChampionID
			var role = gameParticipant.Timeline.Lane



			message := fmt.Sprintf("[%s] **%s** just went %s as __%s__, looks like %s",
				role,
				sd.summoner.Name,
				statline,
				champid,
				getGameStatus(gameParticipant.Stats.Kills, gameParticipant.Stats.Deaths, gameParticipant.Stats.Win))

			_, err = botMain.discord.ChannelMessageSend(sd.reportingChannel.ID, message)
			if err != nil {
				log.WithError(err).WithField("channel", sd.reportingChannel.Name).Errorf("Couldn't post message to channel: %s", message)
				continue
			}

			// Update the last game inspected so we don't repeat ourselves
			sd.Lock()
			sd.lastGameReported = currentGame
			sd.Unlock()
		}
	}
}
