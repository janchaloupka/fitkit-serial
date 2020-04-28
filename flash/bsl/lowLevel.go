package bsl

import (
	"encoding/binary"
	"log"
	"time"

	"go.bug.st/serial.v1"
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
	constErrCom   = "Unspecific error"
	constErrRxNAK = "NAK received (wrong password?)"
	//constErrCmdNotCompleted   = "Command did not send ACK: indicates that it didn't complete correctly"
	constErrCmdFailed   = "Command failed, is not defined or is not allowed"
	constErrBslSync     = "Bootstrap loader synchronization error"
	constErrFrameNumber = "Frame sequence number error."
)

// Connection Otevřené spojení sériové linky
type Connection serial.Port

func sleepMs(millis time.Duration) {
	time.Sleep(millis * time.Millisecond)
}

// Open Otevřít nové sériové spojení s FITkitem v módu programování
func Open(port string) Connection {
	conn, err := serial.Open(port, &serial.Mode{
		BaudRate: 9600,
		DataBits: 8,
		Parity:   serial.EvenParity,
		StopBits: serial.OneStopBit,
	})

	if err != nil {
		log.Fatal(err)
		return nil
	}

	setRstPin(conn, true)
	setTestPin(conn, true)
	flushInput(conn)
	flushOutput(conn)

	return conn
}

// Close Uzavřít spojení a resetovat FITkit
func Close(conn Connection) {
	setRstPin(conn, false)
	setTestPin(conn, false)

	err := conn.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func write(conn Connection, bytes []byte) int {
	n, err := conn.Write(bytes)
	if err != nil {
		log.Fatal(err)
	}

	return n
}

func read(conn Connection, len int) []byte {
	buff := make([]byte, len)

	n, err := conn.Read(buff)
	if err != nil {
		log.Fatal(err)
	}

	return buff[:n]
}

func flushInput(conn Connection) {
	err := conn.ResetInputBuffer()

	if err != nil {
		log.Fatal(err)
	}
}

func flushOutput(conn Connection) {
	err := conn.ResetOutputBuffer()

	if err != nil {
		log.Fatal(err)
	}
}

// Nastaví RST/NMI pin na požadovanou hodnotu
func setRstPin(conn Connection, level bool) {
	err := conn.SetDTR(level)
	sleepMs(10)

	if err != nil {
		log.Fatal(err)
	}
}

// Nastaví TEST pin na požadovanou hodnotu
func setTestPin(conn Connection, level bool) {
	err := conn.SetRTS(level)
	sleepMs(10)

	if err != nil {
		log.Fatal(err)
	}
}

// BslReset Nastaví BSL sekvenci na RST a TEST pinech
// invokeBsl nastavuje, zda se použíje kompletní sekvence nebo
// jen nastavení RST pinu
func BslReset(conn Connection, invokeBsl bool) {
	setRstPin(conn, true)
	setTestPin(conn, true)
	sleepMs(250)

	setRstPin(conn, false)

	if invokeBsl {
		setTestPin(conn, true)
		setTestPin(conn, false)
		setTestPin(conn, true)
		setTestPin(conn, false)
		setRstPin(conn, true)
		setTestPin(conn, true)
	} else {
		setRstPin(conn, true)
	}

	sleepMs(250)
	flushInput(conn)
}

// BslSync Odešle synchronizační znak a očekává potvrzující znak.
// Pokud je wait true, je opakováno dokud se to nepodaří (do nekonečna)
func BslSync(conn Connection, wait bool) {
	for loopcnt := 3; wait || loopcnt >= 0; loopcnt-- {
		flushInput(conn)

		write(conn, []byte{constBslSync})
		bytes := read(conn, 1)

		if len(bytes) >= 1 && bytes[0] == constDataAck {
			// Synchronizace úspěšná
			return
		}
	}

	log.Fatal(constErrBslSync)
	//return errors.New(constErrBslSync)
}

// BslTxRx Odešle příkaz cmd do bootloaderu a vrátí odpověď (nebo nil)
// addr - počáteční adresa
// length - délka
// blkout - další data
// wait - viz bslSync()
func BslTxRx(conn Connection, cmd byte, addr uint16, length uint16, blkout []byte, wait bool) []byte {
	if cmd == constBslTxBLK {
		if (addr % 2) != 0 {
			addr--
			blkout = append([]byte{0xFF}, blkout...)
			length++
		}

		if (length % 2) != 0 {
			blkout = append(blkout, []byte{0xFF}...)
			length++
		}
	} else if cmd == constBslRxBLK {
		if (addr % 2) != 0 {
			addr--
			length++
		}

		if (length % 2) != 0 {
			length++
		}
	}

	dataOut := make([]byte, 4)
	binary.LittleEndian.PutUint16(dataOut[0:2], addr)
	binary.LittleEndian.PutUint16(dataOut[2:4], length)
	dataOut = append(dataOut, blkout...)

	BslSync(conn, wait)

	rxFrame := ComTxRx(conn, cmd, dataOut, length)
	if rxFrame != nil {
		return rxFrame[4:] // Pouze data bez [hdr,null,len,len]
	}

	return nil
}

// ComTxRx Sends the command cmd with the data given in dataOut to the
// microcontroller and expects either an acknowledge or a frame
// with result from the microcontroller.  The results are stored
// in dataIn (if not a NULL pointer is passed).
// In this routine all the necessary protocol stuff is handled.
// Returns zero if the function was successful. TODO přepsat dokumentaci
func ComTxRx(conn Connection, cmd byte, dataOut []byte, length uint16) []byte {
	if (length % 2) != 0 {
		dataOut = append(dataOut, []byte{0xFF}...)
		length++
	}

	txFrame := make([]byte, 4)
	txFrame[0] = constDataFrame // TODO seqNo
	txFrame[1] = cmd
	txFrame[2] = byte(len(dataOut))
	txFrame[3] = byte(len(dataOut))

	// TODO seqNo = (seqNo + 1) % constMaxFrameCount

	txFrame = append(txFrame, dataOut...)
	checksum := calcChecksum(txFrame, length+4)
	txFrame = append(txFrame, []byte{byte(checksum), byte(checksum >> 8)}...)

	//accessAddr := uint16(0x0212+(checksum^0xffff)) & 0xfffe // 0x0212: Address of wCHKSUM
	//TODO

	return nil
}

func calcChecksum(data []byte, length uint16) uint16 {
	checksum := uint16(0)

	for i := uint16(0); i < length/2; i++ {
		checksum = checksum ^ (uint16(data[i*2]) | (uint16(data[i*2+1]) << 8))
	}

	return checksum ^ 0xFFFF
}
