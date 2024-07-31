package main

import (
	"database/sql"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaiserofnone/dmv/bot"

	discord "github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	DBPath    string
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

	bot.ReplyVisible(s, i, "Pong!")
}

func main() {
	logf, err := os.OpenFile("bot.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		logger.Fatalf("Failed to open database: %v", err)
	}
	userConfigs, err := NewUserConfigManager(logger, db)
	if err != nil {
		logger.Fatalf("Failed to create user config manager: %v", err)
	}
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
