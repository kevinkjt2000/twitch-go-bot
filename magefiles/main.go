package main

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

/* To-Do List
See if this project is usable as a plugin framework https://github.com/samwho/streamdeck
Learn how streamdeck's websocket is connected to by loooking at https://github.com/lornajane/streamdeck-tricks
*/

// Runs the bot.
func Run() error {
	fmt.Println("Building...")
	if err := sh.Run("go", "build", "-o", "bot", "cmd/bot/main.go"); err != nil {
		return err
	}
	fmt.Println("Running...")
	return sh.RunV("./bot")
}
