package main

import (
	"database/sql"
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
	db          *sql.DB
	UserConfigs map[string]UserConfig
}

func NewUserConfigManager(logger *log.Logger, db *sql.DB) (*UserConfigManager, error) {
	err := db.Ping()
	if err != nil {
		return nil, err
	}
	return &UserConfigManager{
		logger:      logger,
		db:          db,
		UserConfigs: map[string]UserConfig{},
	}, nil
}

func (uc *UserConfigManager) UpdateUserConfig(tx *sql.Tx, user string, cfg UserConfig) error {
	uc.UserConfigs[user] = cfg
	_, err := tx.Exec("INSERT OR REPLACE INTO user_configs (id, timezone) VALUES (?, ?)", user, cfg.Timezone.String())
	return err
}

func (uc *UserConfigManager) GetUserConfig(tx *sql.Tx, user string) (UserConfig, error) {
	if cfg, ok := uc.UserConfigs[user]; ok {
		return cfg, nil
	}
	var timezone string
	err := tx.QueryRow("SELECT timezone FROM user_configs WHERE id = ?", user).Scan(&timezone)
	if err == sql.ErrNoRows {
		return UserConfig{}, nil
	}
	if err != nil {
		return UserConfig{}, err
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return UserConfig{}, err
	}
	cfg := UserConfig{
		Timezone: loc,
	}
	uc.UserConfigs[user] = cfg
	return cfg, nil
}

// Commands

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
			uc.SetTimezoneHandler(s, i, setopts)
		}
		if getopts, ok := cmd["get"]; ok {
			uc.GetTimezoneHandler(s, i, getopts)
		}
	}
}

func (uc *UserConfigManager) SetTimezoneHandler(
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
	tx, err := uc.db.Begin()
	if err != nil {
		bot.ReplyEphemeral(s, i, "Internal error, please notify the bot owner.")
		uc.logger.Printf("Failed to start transaction: %v", err)
		return
	}
	userConfig, err := uc.GetUserConfig(tx, user)
	if err != nil {
		bot.ReplyEphemeral(s, i, "Internal error, please notify the bot owner.")
		uc.logger.Printf("Failed to get user config: %v", err)
		tx.Rollback()
		return
	}
	userConfig.Timezone = location
	err = uc.UpdateUserConfig(tx, user, userConfig)
	if err != nil {
		bot.ReplyEphemeral(s, i, "Internal error, please notify the bot owner.")
		uc.logger.Printf("Failed to update user config: %v", err)
		tx.Rollback()
		return
	}
	err = tx.Commit()
	if err != nil {
		bot.ReplyEphemeral(s, i, "Internal error, please notify the bot owner.")
		uc.logger.Printf("Failed to commit transaction: %v", err)
		return
	}
	bot.ReplyEphemeral(s, i, fmt.Sprintf("Timezone set to %s", location.String()))
}

func (uc *UserConfigManager) GetTimezoneHandler(
	s *discord.Session,
	i *discord.InteractionCreate,
	data *discord.ApplicationCommandInteractionDataOption) {

	user := i.Interaction.Member.User.ID
	tx, err := uc.db.Begin()
	if err != nil {
		bot.ReplyEphemeral(s, i, "Internal error, please notify the bot owner.")
		uc.logger.Printf("Failed to start transaction: %v", err)
		return
	}
	userCfg, err := uc.GetUserConfig(tx, user)
	if err != nil {
		bot.ReplyEphemeral(s, i, "Internal error, please notify the bot owner.")
		uc.logger.Printf("Failed to get user config: %v", err)
		tx.Rollback()
		return
	}
	tx.Commit()
	if userCfg.Timezone == nil {
		bot.ReplyEphemeral(s, i, "You don't have a timezone set")
		return
	}
	tzname := userCfg.Timezone.String()
	bot.ReplyEphemeral(s, i, fmt.Sprintf("Your timezone is %s", tzname))
}
