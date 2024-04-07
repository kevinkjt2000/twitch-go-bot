package main

import (
	"github.com/magefile/mage/sh"
)

// Runs the bot.
func Run() error {
	if err := sh.Run("go", "build", "-o", "bot", "cmd/bot/main.go"); err != nil {
		return err
	}
	return sh.RunV("./bot")
}
