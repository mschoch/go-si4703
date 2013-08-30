package main

import (
	"fmt"

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
	d.SetVolume(10)

	fmt.Printf("trying to tune 101.1\n")
	d.SetChannel(1011)
	fmt.Printf("tuned\n")
	fmt.Printf("%v\n", d)
}
