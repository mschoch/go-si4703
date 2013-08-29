//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package si4703

import (
	"bytes"
	"encoding/binary"
	"log"

	"bitbucket.org/gmcbay/i2c"
)

const I2C_ADDR = 0x10

// register names
const (
	DEVICEID uint16 = iota
	CHIPID
	POWERCFG
	CHANNEL
	SYSCONFIG1
	SYSCONFIG2
	UNUSED6
	UNUSED7
	UNUSED8
	UNUSED9
	STATUSRSSI
	READCHAN
	RDSA
	RDSB
	RDSC
	RDSD
)

// powercfg
const SMUTE uint16 = 15
const DMUTE uint16 = 14
const SKMODE uint16 = 10
const SEEKUP uint16 = 9
const SEEK uint16 = 8

// channel
const TUNE uint16 = 15

// sysconfig1
const RDS uint16 = 12
const DE uint16 = 11

// sysconfig2
const SPACE1 uint16 = 5
const SPACE0 uint16 = 4

// statusrssi
const RDSR uint16 = 15
const STC uint16 = 14
const SFBL uint16 = 13
const AFCRL uint16 = 12
const RDSS uint16 = 11
const STEREO uint16 = 8

type Device struct {
	bus       *i2c.I2CBus
	busNum    byte
	addr      byte
	registers []uint16
}

func (d *Device) Init(busNum byte) (err error) {
	return d.InitCustomAddr(I2C_ADDR, busNum)
}

func (d *Device) InitCustomAddr(addr, busNum byte) (err error) {
	if d.bus, err = i2c.Bus(busNum); err != nil {
		return
	}

	d.busNum = busNum
	d.addr = addr

	d.readRegisters()

	return
}

func (d *Device) readRegisters() {
	var data []byte
	var err error
	if data, err = d.bus.ReadByteBlock(d.addr, 0, 32); err != nil {
		return
	}

	counter := 0
	for x := 0x0A; ; x++ {
		if x == 0x10 {
			x = 0
		}
		p := bytes.NewBuffer(data[counter : counter+2])
		err = binary.Read(p, binary.BigEndian, d.registers[x])
		if err != nil {
			return
		}
		counter = counter + 2
		if x == 0x09 {
			break
		}
	}

	log.Printf("read bytes: %v", data)

	log.Printf("self: %v", d)
}

func (d *Device) updateRegisters() {

}
