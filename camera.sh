#!/usr/bin/env bash
set -euo pipefail

#    /camera create – Creates a camera at the Player’s position, using the Player’s head orientation angles
#    /camera create <x, y, z, pitch, yaw> – Creates a camera at user defined coordinates
#    /camera list – Lists all the cameras currently in world
#    /camera remove <id> – Removes the specified camera
#    /camera remove all – Removes all cameras
#    /camera switch <id> – Switches to the specified camera
#    /camera back – Switches back to the Player

gt_window=$(xdotool search --name "GT:")

remove_cameras() {
    # be sure to subtract 1 for 0-based indexing
    local num_cameras=$(($1 - 1))
    for i in $(seq 0 $num_cameras); do
        chat "/camera remove $i"
    done
}

hive_cameras() {
    local num_cameras=12
    remove_cameras $num_cameras
    chat "/effect GleamingPail 14 99999" # 27 hours of invisibility
    chat "/camera create 27.3 70.835 -42.3 25 -128"       # LV shrine room
    chat "/camera create 92.39 72.2 -100.39 30 46.7457"   # THE HIVE - steam tank view
    chat "/camera create 79.04411 72.2 -68.3 30 143.4956" # THE HIVE - LV machines
    chat "/camera create 65.3 67.835 -63.7 25 -44.152161" # Kitchen hallway
    chat "/camera create 72.16757 62.2 -34.4 30 149.347"  # Kitchen
    chat "/camera create 44.2375 62.19 -26.87 25 -126.05" # Inventory room
    chat "/camera create 36.7 60.2 -30.937 25 110.347"    # Thaumcraft balcony
    chat "/camera create 38.7 93 -64.29 30 -54.057"       # Courtyard -> botania
    chat "/camera create 43.85 77 -20.35 5 -163"          # Courtyard -> BBFs
    chat "/camera create 38.9 94 -72.68 31 -145.85"       # BBFs
    chat "/camera create 38.3 98 -105.68 31 139.747"      # Crops
    chat "/camera create 46.34 76.855 -42.71 26 135.996"  # Oil processing
    loop_cameras $num_cameras
}

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

anti_idle() {
    xdotool keydown --window $gt_window a
    sleep 0.1
    xdotool keydown --window $gt_window e
    sleep 0.1
    xdotool keyup --window $gt_window a
    sleep 0.1
    xdotool keyup --window $gt_window e
    sleep 0.1
}

loop_cameras() {
    local num_cameras=$1
    while true; do
        for i in $(seq 0 $num_cameras); do
            anti_idle
            chat "/camera switch $i"
            sleep 120
        done
    done
}

activate_chat() {
    xdotool windowactivate "$gt_window"
    sleep 0.1
    xdotool key --window $gt_window t
    sleep 0.1
    xdotool key --window $gt_window Escape
    sleep 0.1
    xdotool key --window $gt_window t
    sleep 0.1
}

chat() {
    activate_chat
    xdotool type --window $gt_window "$1"
    sleep 0.2
    xdotool key --window $gt_window Return
    sleep 0.1
}

public_cameras
chat "/camera back"

# You are viewing the Security Booth™ of the private Hive Optimization Run. These are various camera positions showing 2 minutes at a time from the actual server. If you want to see more, be sure to follow the users mentioned in title or checkout the community server https://youtu.be/4GiuDQ05Ug4?si=cFCKa0uUusDWsE2T

# You are viewing the Security Booth™ of the public Hive Optimization Run. These are various camera positions showing 2 minutes at a time from the actual server. If you want to join, checkout the information on Diddy's discord https://discord.gg/diddyshive