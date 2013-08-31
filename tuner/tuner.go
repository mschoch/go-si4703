package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mschoch/go-si4703"
)

func main() {
	var i2cbus byte = 1
	d := new(si4703.Device)

	err := d.Init(i2cbus)

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("init done\n")

	fmt.Printf("setting volume\n")
	d.SetVolume(5)

	fmt.Printf("trying to tune 101.1\n")
	d.SetChannel(1011)
	fmt.Printf("tuned\n")
	fmt.Printf("%v\n", d)

	// fmt.Printf("disabling soft mute")
	// d.DisableSoftMute()

	fmt.Printf("disabled mute")
	d.DisableMute()

	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.Replace(line, "\n", "", -1)

		command := strings.SplitN(line, " ", 2)
		switch command[0] {
		case "quit":
			os.Exit(0)
		case "volume":
			if len(command) > 1 {
				val, err := strconv.Atoi(command[1])
				if err != nil || val < 0 || val > 15 {
					fmt.Printf("Invalid volume level, must be (0-15)\n")
					continue
				}
				d.SetVolume(uint16(val))
			} else {
				fmt.Printf("specify a volume level (0-15)\n")
			}
		case "mute":
			if len(command) > 1 {
				val := command[1]
				if val == "on" {
					d.EnableMute()
				} else if val == "off" {
					d.DisableMute()
				} else {
					fmt.Printf("Invalid setting, must be (on/off)\n")
				}
			} else {
				fmt.Printf("specify setting (on/off)\n")
			}
		case "tune":
			if len(command) > 1 {
				val, err := strconv.ParseFloat(command[1], 64)
				if err != nil {
					fmt.Printf("Invalid frequence\n")
				} else {
					val = val * 10
					freqint := uint16(val)
					d.SetChannel(freqint)
				}
			} else {
				fmt.Printf("specify frequency in MHz\n")
			}
		case "help":
			fmt.Printf("Valid commands are: quit, volume, mute, tune\n")
		default:
			fmt.Printf("Unknown command: `%s`\n", command[0])
		}

	}
}
