## go-si4703

A go library to interact with the [Si4703](https://www.sparkfun.com/datasheets/BreakoutBoards/Si4702-03-C19-1.pdf) FM Tuner (with RDS support).  

NOTE:  This has only been tested on a Raspberry Pi.

### Connections

Connect SDIO/SCLK/3.3V/GND as you would any other I²C/2-wire device.
Connect the RST to GPIO 23 on the rpi (needed to put device into 2-wire mode)

### Examples

There is one example app allowing you to interact with the tuner.

    $ sudo ./tuner 

