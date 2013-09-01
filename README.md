## go-si4703

A go library to interact with the [Si4703](https://www.sparkfun.com/datasheets/BreakoutBoards/Si4702-03-C19-1.pdf) FM Tuner (with RDS support).  

NOTE:  This has only been tested on a Raspberry Pi.

### Connections

* SDIO/SCLK/3.3V/GND as you would any other I²C/2-wire device.
* RST to GPIO 23 on the rpi (needed to put device into 2-wire mode)

### Examples

There is one example app allowing you to interact with the tuner.

    $ sudo ./tuner
    tune 101.1
    2013/09/01 16:49:22 Attempting to tune
    2013/09/01 16:49:22 Tuned to Channel: 101.100000MHz
    mute off
    volume 5
    seek down
    2013/09/01 16:49:39 Seeking DOWN
    2013/09/01 16:49:39 Seeked to Channel: 100.300000MHz
    status
    --------------------------------------------------------------------------------
    Part Number: Si4702/03
    Manufacturer: 0x242
    Chip Version: Rev C
    Device: Si4702 (on)
    Firmware Version 19
    Soft Mute: Enabled
    Mute: Disabled
    Force Mono: Disabled
    RDS Mode: Standard
    Seek Mode: Wrap
    Seek Direction: Down
    Seek: Disabled
    Power-Up Disable: Default
    Power-Up Enable: On
    Tune: Disabled
    Tune Channel: 101.100000MHz
    RDS Interrupt: Disabled
    Seek/Tune Complete Interrupt: Disabled
    RDS: Enabled
    De-emphasis: 75μs
    AGC: Disabled
    Stereo/Mono Blend Adjustment: 31–49 RSSI dBµV (default)
    RDS Ready: No RDS group ready
    Seek/Tune Complete: Not complete
    Seek Fail/Band Limit: Seek successful
    AFC Rail: AFC not railed
    RDS Synchronized: RDS decoder not synchronized
    Stereo/Mono: Stereo
    RSSI: 54dBµV
    Channel: 100.300000MHz
    A: ÿ
    B: 
    C: 
    D: 
    --------------------------------------------------------------------------------

    quit



