// The bot package exposes an opaque struct to handle discord commands fairly similarly
// to net/http handles requests.
//
// The lifecycle for a bot is New -> RegisterHandler (s) -> Start
// this is because handlers are added on startup.
package bot

import (
	"log"

	discord "github.com/bwmarrin/discordgo"
)

type Config struct {
	Token         string   // Secret token, do not commit it to gh you dingus
	GuildIds      []string // List of guilds where the commands will be available
	ApplicationId string
}

type Bot struct {
	logger          *log.Logger
	commandHandlers map[string]CommandHandler
	commands        []*discord.ApplicationCommand
	cfg             Config
	discordClient   *discord.Session
	stop            chan bool
}

func NewBot(cfg Config, logger *log.Logger) (*Bot, error) {
	client, err := discord.New("Bot " + cfg.Token)
	if err != nil {
		return nil, err
	}
	return &Bot{
		cfg:             cfg,
		logger:          logger,
		commandHandlers: map[string]CommandHandler{},
		discordClient:   client,
		stop:            make(chan bool),
	}, nil
}

func (b *Bot) Start() error {
	b.discordClient.AddHandler(func(s *discord.Session, r *discord.Ready) {
		b.logger.Printf("Logged in as %s", r.User.String())
	})
	err := b.registerCommands()
	if err != nil {
		return err
	}
	err = b.discordClient.Open()
	if err != nil {
		return err
	}
	select {
	case <-b.stop:
		return nil
	}
}

func (b *Bot) Shutdown() error {
	err := b.discordClient.Close()
	b.stop <- true
	return err
}

func (b *Bot) RegisterHandler(descriptor *discord.ApplicationCommand, f CommandHandler) {
	b.commands = append(b.commands, descriptor)
	b.commandHandlers[descriptor.Name] = f
}

func (b *Bot) registerCommands() error {
	b.discordClient.AddHandler(func(s *discord.Session, i *discord.InteractionCreate) {
		if i.Type != discord.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		b.dispatchCommand(s, i, data)
	})
	for _, guildId := range b.cfg.GuildIds {
		b.logger.Printf("Registering commands for %v", guildId)
		_, err := b.discordClient.ApplicationCommandBulkOverwrite(b.cfg.ApplicationId, guildId, b.commands)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Bot) dispatchCommand(s *discord.Session,
	i *discord.InteractionCreate,
	data discord.ApplicationCommandInteractionData) {
	command, found := b.commandHandlers[data.Name]
	if !found {
		return
	}
	command(s, i, data)
}

type CommandHandler = func(
	s *discord.Session,
	i *discord.InteractionCreate,
	data discord.ApplicationCommandInteractionData)

func ReplyEphemeral(s *discord.Session, i *discord.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseChannelMessageWithSource,
		Data: &discord.InteractionResponseData{
			Content: message,
			Flags:   discord.MessageFlagsEphemeral,
		},
	})
}

func ReplyVisible(s *discord.Session, i *discord.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseChannelMessageWithSource,
		Data: &discord.InteractionResponseData{
			Content: message,
		},
	})
}

func CollectOptions(opts []*discord.ApplicationCommandInteractionDataOption) map[string]*discord.ApplicationCommandInteractionDataOption {
	rv := map[string]*discord.ApplicationCommandInteractionDataOption{}
	for _, opt := range opts {
		rv[opt.Name] = opt
	}
	return rv
}
