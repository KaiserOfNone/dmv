package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kaiserofnone/dmv/bot"

	discord "github.com/bwmarrin/discordgo"
	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	BotConfig bot.Config `toml:"bot"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	doc, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = toml.Unmarshal(doc, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

var PongDescriptor = &discord.ApplicationCommand{
	Name:        "pong",
	Description: "Replies with pong",
}

func PongHandler(
	s *discord.Session,
	i *discord.InteractionCreate,
	data discord.ApplicationCommandInteractionData) {
	s.InteractionRespond(i.Interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseChannelMessageWithSource,
		Data: &discord.InteractionResponseData{
			Content: "Pong!",
		},
	})
}

type UserConfig struct {
	Timezone *time.Location
}

type UserConfigManager struct {
	logger      *log.Logger
	UserConfigs map[string]UserConfig
}

func LoadUserConfigs(logger *log.Logger) UserConfigManager {
	return UserConfigManager{
		logger:      logger,
		UserConfigs: map[string]UserConfig{},
	}
}

var ConfigureDescriptor = &discord.ApplicationCommand{
	Name:        "configure",
	Description: "Configure User settings",
	Options: []*discord.ApplicationCommandOption{
		{
			Name:        "timezone",
			Description: "Timezone related commands",
			Type:        discord.ApplicationCommandOptionSubCommandGroup,
			Options: []*discord.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Sets the user timezone, use IANA timezone format (continent/city)",
					Type:        discord.ApplicationCommandOptionSubCommand,
					Options: []*discord.ApplicationCommandOption{
						{
							Name:        "timezone",
							Description: "IANA timezone name",
							Type:        discord.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
	},
}

func (uc *UserConfigManager) ConfigureHandler(
	s *discord.Session,
	i *discord.InteractionCreate,
	data discord.ApplicationCommandInteractionData) {
	if data.Options[0].Name == "timezone" {
		uc.ConfigureTimezoneHandler(s, i, data.Options[0].Options[0])
	}
}

func (uc *UserConfigManager) ConfigureTimezoneHandler(
	s *discord.Session,
	i *discord.InteractionCreate,
	data *discord.ApplicationCommandInteractionDataOption) {
	user := i.Interaction.Member.User.ID
	if data.Name == "set" {
		tzname := (data.Options[0].Value).(string)
		location, err := time.LoadLocation(tzname)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discord.InteractionResponse{
				Type: discord.InteractionResponseChannelMessageWithSource,
				Data: &discord.InteractionResponseData{
					Content: fmt.Sprintf("Invalid location: %s: %v", location, err),
					Flags:   discord.MessageFlagsEphemeral,
				},
			})
			return
		}
		userConfig := uc.UserConfigs[user]
		userConfig.Timezone = location
		uc.UserConfigs[user] = userConfig
		s.InteractionRespond(i.Interaction, &discord.InteractionResponse{
			Type: discord.InteractionResponseChannelMessageWithSource,
			Data: &discord.InteractionResponseData{
				Content: fmt.Sprintf("Timezone set to %s", location.String()),
				Flags:   discord.MessageFlagsEphemeral,
			},
		})
	}
}

func main() {
	logf, err := os.Create("bot.log")
	if err != nil {
		panic("Failed to create log file")
	}
	defer logf.Close()
	logger := log.New(logf, "", log.LstdFlags)
	log.Print("Starting bot")
	path := "./bot.toml"
	logger.Printf("Loading up config at %s", path)
	cfg, err := LoadConfig(path)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	userConfigs := LoadUserConfigs(logger)
	discordBot, err := bot.NewBot(cfg.BotConfig, logger)
	discordBot.RegisterHandler(PongDescriptor, PongHandler)
	discordBot.RegisterHandler(ConfigureDescriptor, userConfigs.ConfigureHandler)
	go func() {
		err = discordBot.Start()
		if err != nil {
			logger.Fatalf("Fatal error: %v", err)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	discordBot.Shutdown()
	logger.Printf("Shutting down")
}
