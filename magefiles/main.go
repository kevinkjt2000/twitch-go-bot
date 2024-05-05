package main

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

// Runs the bot.
func Run() error {
	fmt.Println("Building...")
	if err := sh.Run("go", "build", "-o", "bot", "cmd/bot/main.go"); err != nil {
		return err
	}
	fmt.Println("Running...")
	return sh.RunV("./bot")
}
