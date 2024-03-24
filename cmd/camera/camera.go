package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"
)

var hiveCameras = []Camera{
	{"27.3 70.835 -42.3 25 -128"},       // LV shrine room
	{"92.39 72.2 -100.39 30 46.7457"},   // THE HIVE - steam tank view
	{"79.04411 72.2 -68.3 30 143.4956"}, // THE HIVE - LV machines
	{"65.3 67.835 -63.7 25 -44.152161"}, // Kitchen hallway
	{"72.16757 61.5 -34.4 27 149.347"},  // Kitchen
	{"44.2375 62.19 -26.87 25 -126.05"}, // Inventory room
	{"36.7 60.2 -30.937 25 110.347"},    // Thaumcraft balcony
	{"38.7 93 -64.29 30 -54.057"},       // Courtyard -> botania
	{"27.3 85.86 8.5 20 -144.65"},       // Courtyard -> MBBF & bees
	{"46.34 76.855 -42.71 26 135.996"},  // Oil processing
	{"201 103 -98 18 -118.55"},          // Benzene
	{"166.7 89 -92.7 7 48.2"},           // Chandelier storage
	{"174 111.3 -47.88 17 -129.36"},     // Entrance - ancient tree
	{"44 94 -127 12 111.5"},             // MBBF -> crops
	{"18.8 70.7 -127.4 15 146.04"},      // Crops B1
}

const (
	keystrokeDelayMs  = "2"
	delayAfterCommand = 80 * time.Millisecond
	delayBetweenSwaps = 3 * time.Second
)

type Camera struct {
	Location string
}

/*
#    /camera create – Creates a camera at the Player’s position, using the Player’s head orientation angles
#    /camera create <x, y, z, pitch, yaw> – Creates a camera at user defined coordinates
#    /camera list – Lists all the cameras currently in world
#    /camera remove <id> – Removes the specified camera
#    /camera remove all – Removes all cameras
#    /camera switch <id> – Switches to the specified camera
#    /camera back – Switches back to the Player
*/

func (c *Camera) TeleportLocation() string {
	pieces := strings.Split(c.Location, " ")
	return pieces[0] + " " + pieces[1] + " " + pieces[2]
}

/*
public_cameras() {
    local num_cameras=8
    remove_cameras $num_cameras
    chat "/camera create -245 78.2 -719.18 25 -126.45"     # trophy heads
    chat "/camera create -250.5 76.7 -748 20 142.64"       # hive mbbf
    chat "/camera create -316.74 69.3 -787.77 20 -15.75"   # trampled crops that aren't autofarmed
    chat "/camera create -108 101 -722 30 143.24"          # auto crops
    chat "/camera create -280.5 70 -719 20 -28"            # thaumic infusion
    chat "/camera create -174 69 -693.6 20 58"             # ore processing and main machine area
    chat "/camera create -171 67 -696 20 -112.96"          # "often visited"
    chat "/camera create -257.86 75 -642.4 25 52.63"       # bees
    loop_cameras $num_cameras
}
*/

// You are viewing the Security Booth™ of the private Hive Optimization Run. These are various camera positions showing 2 minutes at a time from the actual server. If you want to see more, be sure to follow the users mentioned in title or checkout the community server https://youtu.be/4GiuDQ05Ug4?si=cFCKa0uUusDWsE2T

// You are viewing the Security Booth™ of the public Hive Optimization Run. These are various camera positions showing 2 minutes at a time from the actual server. If you want to join, checkout the information on Diddy's discord https://discord.gg/diddyshive

func getGtWindow() string {
	cmd := exec.Command("xdotool", "search", "--name", "GT:")
	stdout, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(stdout))
}

func antiIdle(gtWindow string) {
	exec.Command("xdotool", "keydown", "--window", gtWindow, "a").Run()
	time.Sleep(delayAfterCommand)
	exec.Command("xdotool", "keydown", "--window", gtWindow, "e").Run()
	time.Sleep(delayAfterCommand)
	exec.Command("xdotool", "keyup", "--window", gtWindow, "a").Run()
	time.Sleep(delayAfterCommand)
	exec.Command("xdotool", "keyup", "--window", gtWindow, "e").Run()
	time.Sleep(delayAfterCommand)
}

func setupCameras(gtWindow string, cameras []Camera) {
	for i, cam := range cameras {
		cam.TeleportLocation()
		chat(gtWindow, fmt.Sprintf("/camera remove %d", i))
		chat(gtWindow, fmt.Sprintf("/camera create %s", cam.Location))
	}
}

func chat(gtWindow string, msg string) {
	exec.Command("xdotool", "windowactivate", gtWindow).Run()
	time.Sleep(delayAfterCommand)
	exec.Command("xdotool", "key", "--window", gtWindow, "t").Run()
	time.Sleep(delayAfterCommand)
	exec.Command("xdotool", "key", "--window", gtWindow, "Escape").Run()
	time.Sleep(delayAfterCommand)
	exec.Command("xdotool", "key", "--window", gtWindow, "t").Run()
	time.Sleep(delayAfterCommand)

	exec.Command("xdotool", "type", "--delay", keystrokeDelayMs, "--window", gtWindow, msg).Run()
	time.Sleep(delayAfterCommand)
	exec.Command("xdotool", "key", "--window", gtWindow, "Enter").Run()
	time.Sleep(delayAfterCommand)
}

func loopCameras(ctx context.Context, gtWindow string, cameras []Camera) {
	chat(gtWindow, "/effect GleamingPail 14 99999") // 27 hours of invisibility
	for i, cam := range cameras {
		// anti_idle(gtWindow)
		chat(gtWindow, "/tp "+cam.TeleportLocation())
		chat(gtWindow, fmt.Sprintf("/camera switch %d", i))

		select {
		case <-ctx.Done():
			return
		case <-time.After(delayBetweenSwaps):
		}
	}
}

func main() {
	gtWindow := getGtWindow()
	setupCameras(gtWindow, hiveCameras)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	loopCameraCtx, loopCameraCancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-loopCameraCtx.Done():
				return
			default:
				loopCameras(loopCameraCtx, gtWindow, hiveCameras)
			}
		}
	}()

	<-interrupt
	chat(gtWindow, "/camera back")
	loopCameraCancel()
}
