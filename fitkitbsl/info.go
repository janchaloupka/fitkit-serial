package fitkitbsl

import (
	"errors"
	"time"
)

// Info page contains 264B and is located inside external FLASH at page 0
// 	Structure
// 		MCU [0 - 127]
// 		+ 0      01 - programmed, 00 - unprogramed
// 		+ 1-3    3B flash write counter
// 		+ 4-19   MD5 digest of HEX file
// 		+ 20-23  file modification timestamp
// 		+ 24-27  programming timestamp
// 		+ 28-127 Description (C-like string, up to 99 chars)
// 		FPGA BIN [128 - 255]
// 		+ 0      01 - programmed, 00 - unprogramed
// 		+ 1-3    3B flash write counter
// 		+ 4-19   MD5 digest of BIN file
// 		+ 20-23  file modification timestamp
// 		+ 24-27  programming timestamp
// 		+ 28-127 Description (C-like string, up to 99 chars)
// 		ID [256-263]
// 		+ 0-7    string "FINFO v1"
type Info struct {
	version int
	data    [264]byte
}

// NewInfo - Initialize info instance
func NewInfo(rawData []byte) (*Info, error) {
	if len(rawData) != 264 {
		return nil, errors.New("Info page data must be exactly 264 bytes in length, got: " + string(len(rawData)))
	}

	i := Info{
		version: 0,
		data:    [264]byte{},
	}

	if string(rawData[256:264]) == "FINFO v1" {
		i.version = 1
		copy(i.data[:], rawData[:264])
	}

	return &i, nil
}

// VersionID - Get version ID
func (i *Info) VersionID() []byte {
	return i.data[256:264]
}

// Convert 4 bytes to timestamp
func (i *Info) byteToTimestamp(data []byte) (uint32, error) {
	if len(data) != 4 {
		return 0, errors.New("Timestamp array must have 4 bytes")
	}

	tst := (uint32(data[0]) << 24) + (uint32(data[1]) << 16)
	tst += (uint32(data[2]) << 8) + uint32(data[3])
	return tst, nil
}

// Convert timestamp to 4 bytes
func (i *Info) timestampToByte(tst uint32) []byte {
	return []byte{
		byte(tst >> 24),
		byte(tst >> 16),
		byte(tst >> 8),
		byte(tst),
	}
}

// GetHexInfo - Parse info about HEX file
func (i *Info) GetHexInfo() (digitest []byte, writeCnt uint32, modTime uint32, flashTime uint32, desc string, err error) {
	digitest = make([]byte, 16) // MD5 digitest
	writeCnt = 0                // Write count
	modTime = 0                 // Modification timestamp
	flashTime = 0               // Flash timestamp
	desc = ""                   // HEX description
	err = nil

	// Is HEX programmed
	if i.data[0] == 0x01 {
		writeCnt = (uint32(i.data[1]) << 16) + (uint32(i.data[2]) << 8) + uint32(i.data[3])
		digitest = i.data[4 : 4+16]
		modTime, err = i.byteToTimestamp(i.data[20:24])
		if err == nil {
			flashTime, err = i.byteToTimestamp(i.data[24:28])
		}

		descEnd := 28
		for ; descEnd < 28+100; descEnd++ {
			if i.data[descEnd] == 0 {
				break
			}
		}

		desc = string(i.data[28:descEnd])
	}

	return digitest, writeCnt, modTime, flashTime, desc, err
}

// GetBinInfo - Parse info about BIN file
func (i *Info) GetBinInfo() (digitest []byte, writeCnt uint32, modTime uint32, flashTime uint32, desc string, err error) {
	digitest = make([]byte, 16) // MD5 digitest
	writeCnt = 0                // Write count
	modTime = 0                 // Modification timestamp
	flashTime = 0               // Flash timestamp
	desc = ""                   // BIN description
	err = nil

	// Is BIN programmed
	if i.data[128] == 0x01 {
		writeCnt = (uint32(i.data[128+1]) << 16) + (uint32(i.data[128+2]) << 8) + uint32(i.data[128+3])
		digitest = i.data[128+4 : 128+4+16]
		modTime, err = i.byteToTimestamp(i.data[128+20 : 128+24])
		if err == nil {
			flashTime, err = i.byteToTimestamp(i.data[128+24 : 128+28])
		}

		descEnd := 128 + 28
		for ; descEnd < 128+28+100; descEnd++ {
			if i.data[descEnd] == 0 {
				break
			}
		}

		desc = string(i.data[128+28 : descEnd])
	}

	return digitest, writeCnt, modTime, flashTime, desc, err
}

// UpdateHexInfo - Update information about HEX file
func (i *Info) UpdateHexInfo(digitest []byte, modTime uint32, desc string) {
	i.data[0] = 0x01

	// Update digitest
	copy(i.data[4:4+16], digitest[0:16])

	// Update counter
	writeCnt := (uint32(i.data[1]) << 16) + (uint32(i.data[2]) << 8) + uint32(i.data[3])
	writeCnt++
	i.data[1] = byte(writeCnt >> 16)
	i.data[2] = byte(writeCnt >> 8)
	i.data[3] = byte(writeCnt >> 0)

	// Update timestamps
	now := time.Now()
	copy(i.data[20:24], i.timestampToByte(modTime))
	copy(i.data[24:28], i.timestampToByte(uint32(now.Unix())))

	// Update description
	for k := 0; k < 100; k++ {
		if k < len(desc) {
			i.data[28+k] = desc[k]
		} else {
			i.data[28+k] = 0
		}
	}
}

// UpdateBinInfo - Update information about BIN file
func (i *Info) UpdateBinInfo(digitest []byte, modTime uint32, desc string) {
	i.data[128] = 0x01

	// Update digitest
	copy(i.data[128+4:128+4+16], digitest[0:16])

	// Update counter
	writeCnt := (uint32(i.data[128+1]) << 16) + (uint32(i.data[128+2]) << 8) + uint32(i.data[128+3])
	writeCnt++
	i.data[128+1] = byte(writeCnt >> 16)
	i.data[128+2] = byte(writeCnt >> 8)
	i.data[128+3] = byte(writeCnt >> 0)

	// Update timestamps
	now := time.Now()
	copy(i.data[128+20:128+24], i.timestampToByte(modTime))
	copy(i.data[128+24:128+28], i.timestampToByte(uint32(now.Unix())))

	// Update description
	for k := 0; k < 100; k++ {
		if k < len(desc) {
			i.data[128+28+k] = desc[k]
		} else {
			i.data[128+28+k] = 0
		}
	}
}

// GetRawData - Return info data
func (i *Info) GetRawData() []byte {
	copy(i.data[256:264], []byte("FINFO v1"))
	return i.data[:]
}
