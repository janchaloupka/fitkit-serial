package mspbsl

import (
	"time"

	"github.com/albenik/go-serial/v2"
)

const (
	// Constants
	constModeSSP = 0
	constModeBSL = 1

	constBslSync       = 0x80
	constBslTxPword    = 0x10
	constBslTxBLK      = 0x12 // Transmit block to boot loader
	constBslRxBLK      = 0x14 // Receive  block from boot loader
	constBslErase      = 0x16 // Erase one segment
	constBslMEras      = 0x18 // Erase complete FLASH memory
	constBslChangeBaud = 0x20 // Change baudrate
	constBslLoadPC     = 0x1A // Load PC and start execution
	constBslTxVersion  = 0x1E // Get BSL version

	// Upper limit of address range that might be modified by
	// BSL checksum bug
	constBslCriticalAddr = 0x0A00

	// Header Definitions
	constCmdFailed = 0x70
	constDataFrame = 0x80
	constDataAck   = 0x90
	constDataNak   = 0xA0

	constQueryPoll     = 0xB0
	constQueryResponse = 0x50

	constOpenConnection = 0xC0
	constAckConnection  = 0x40

	constDefaultTimeout = 1
	constDefaultProlong = 10
	constMaxFrameSize   = 256
	constMaxDataBytes   = 250
	constMaxDataWords   = 125

	constMaxFrameCount = 16

	//Error messages

	ErrReadTimeout = "Serial read timeout"
	ErrCom         = "Unspecific error"
	ErrRxNAK       = "NAK received (wrong password?)"
	ErrCmdFailed   = "Command failed, is not defined or is not allowed"
	ErrBslSync     = "Bootstrap loader synchronization error"
	ErrFrameNumber = "Frame sequence number error."
	//ConstErrCmdNotCompleted   = "Command did not send ACK: indicates that it didn't complete correctly"

	ErrVerifyFailed     = "Verification failed"
	ErrEraseCheckFailed = "Erase check failed"

	constActionProgram    = 0x01 // Mask: program data
	constActionVerify     = 0x02 // Mask: verify data
	constActionEraseCheck = 0x04 // Mask: erase check

	// Max. bytes sent within one frame if parsing a TI TXT file
	// ( >= 16 and == n*16 and <= MAX_DATA_BYTES!)
	constMaxData = 224

	// Cpu types for "change baudrate"

	F1x = "F1x family"
	F2x = "F2x family"
	F4x = "F4x family"
)

func sleepMs(millis time.Duration) {
	time.Sleep(millis * time.Millisecond)
}

func calcChecksum(data []byte, length uint16) uint16 {
	checksum := uint16(0)
	for i := uint16(0); i < length/2; i++ {
		checksum = checksum ^ (uint16(data[i*2]) | (uint16(data[i*2+1]) << 8))
	}

	return checksum ^ 0xFFFF
}

// Known devices list
var deviceIDs = map[uint16]string{
	0x1132: F1x,
	0x1232: F1x,
	0xF112: F1x,
	0xF123: F1x,
	0xF149: F1x,
	0xF169: F1x,
	0xF16C: F1x,
	0xF26F: F2x,
	0xF413: F4x,
	0xF427: F4x,
	0xF439: F4x,
	0xF449: F4x,
}

type baudrateEntry struct {
	addr uint16
	len  uint16
}

// Baudrate list with values for F1x family CPU
var baudrateF1x = map[int]baudrateEntry{
	9600:  {addr: 0x8580, len: 0x0000},
	19200: {addr: 0x86e0, len: 0x0001},
	38400: {addr: 0x87e0, len: 0x0002},
}

// Baudrate list with values for F2x family CPU
var baudrateF2x = map[int]baudrateEntry{
	9600:  {addr: 0x8880, len: 0x0000},
	19200: {addr: 0x8a80, len: 0x0001},
	38400: {addr: 0x8f80, len: 0x0002},
}

// Baudrate list with values for F4x family CPU
var baudrateF4x = map[int]baudrateEntry{
	9600:  {addr: 0x9800, len: 0x0000},
	19200: {addr: 0xb000, len: 0x0001},
	38400: {addr: 0xc800, len: 0x0002},
}

// copy of the patch file provided by TI
// this part is (C) by Texas Instruments
const patch = `@0220
31 40 1A 02 09 43 B0 12 2A 0E B0 12 BA 0D 55 42
0B 02 75 90 12 00 1F 24 B0 12 BA 02 55 42 0B 02
75 90 16 00 16 24 75 90 14 00 11 24 B0 12 84 0E
06 3C B0 12 94 0E 03 3C 21 53 B0 12 8C 0E B2 40
10 A5 2C 01 B2 40 00 A5 28 01 30 40 42 0C 30 40
76 0D 30 40 AC 0C 16 42 0E 02 17 42 10 02 E2 B2
08 02 14 24 B0 12 10 0F 36 90 00 10 06 28 B2 40
00 A5 2C 01 B2 40 40 A5 28 01 D6 42 06 02 00 00
16 53 17 83 EF 23 B0 12 BA 02 D3 3F B0 12 10 0F
17 83 FC 23 B0 12 BA 02 D0 3F 18 42 12 02 B0 12
10 0F D2 42 06 02 12 02 B0 12 10 0F D2 42 06 02
13 02 38 E3 18 92 12 02 BF 23 E2 B3 08 02 BC 23
30 41
q
`

// Instance Zastřešuje metody pro nízkoúrovňovou komunikaci s FITkitem
type Instance struct {
	conn                *serial.Port
	reqNo               byte
	seqNo               byte
	bslMemAccessWarning bool
	Passwd              []byte
	DevID               uint16
	CPUFamily           string
	bslVer              uint16
	patchRequired       bool
	patchLoaded         bool
	byteCtr             int
}
