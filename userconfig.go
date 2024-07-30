package main

import (
	"fmt"
	"log"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/kaiserofnone/dmv/bot"
)

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
				{
					Name:        "get",
					Description: "Gets the user timezone, use IANA timezone format (continent/city)",
					Type:        discord.ApplicationCommandOptionSubCommand,
				},
			},
		},
	},
}

func (uc *UserConfigManager) ConfigureHandler(
	s *discord.Session,
	i *discord.InteractionCreate,
	data discord.ApplicationCommandInteractionData) {
	opts := bot.CollectOptions(data.Options)
	if tzgroup, ok := opts["timezone"]; ok {
		cmd := bot.CollectOptions(tzgroup.Options)
		if setopts, ok := cmd["set"]; ok {
			uc.ConfigureTimezoneHandler(s, i, setopts)
		}
		if getopts, ok := cmd["get"]; ok {
			uc.GetTimezoneHandler(s, i, getopts)
		}
	}
}

func (uc *UserConfigManager) ConfigureTimezoneHandler(
	s *discord.Session,
	i *discord.InteractionCreate,
	data *discord.ApplicationCommandInteractionDataOption) {
	user := i.Interaction.Member.User.ID
	opts := bot.CollectOptions(data.Options)
	tzName := opts["timezone"].Value.(string)
	location, err := time.LoadLocation(tzName)
	if err != nil {
		bot.ReplyEphemeral(s, i, fmt.Sprintf("Invalid location: %s: %v", location, err))
		return
	}
	userConfig := uc.UserConfigs[user]
	userConfig.Timezone = location
	uc.UserConfigs[user] = userConfig
	bot.ReplyEphemeral(s, i, fmt.Sprintf("Timezone set to %s", location.String()))
}

func (uc *UserConfigManager) GetTimezoneHandler(
	s *discord.Session,
	i *discord.InteractionCreate,
	data *discord.ApplicationCommandInteractionDataOption) {
	user := i.Interaction.Member.User.ID
	userCfg, found := uc.UserConfigs[user]
	if !found || userCfg.Timezone == nil {
		bot.ReplyEphemeral(s, i, "You don't have a timezone set")
		return
	}
	tzname := userCfg.Timezone.String()
	bot.ReplyEphemeral(s, i, fmt.Sprintf("Your timezone is %s", tzname))
}
