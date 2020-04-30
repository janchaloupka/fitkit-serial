package mspbsl

import (
	"encoding/binary"
	"errors"
	"log"

	"github.com/albenik/go-serial/v2"
)

// New - Otevřít nové sériové spojení s FITkitem a vytvořit BSL objekt
func New(port string) (*Instance, error) {
	conn, err := serial.Open(
		port,
		serial.WithBaudrate(9600),
		serial.WithDataBits(8),
		serial.WithParity(serial.EvenParity),
		serial.WithStopBits(serial.OneStopBit),
		serial.WithReadTimeout(1000),
	)

	if err != nil {
		return nil, err
	}

	b := Instance{
		conn:                conn,
		reqNo:               0,
		seqNo:               0,
		bslMemAccessWarning: false,
		Passwd:              nil,
		DevID:               0,
		CPUFamily:           "",
		bslVer:              0,
		patchRequired:       false,
		patchLoaded:         false,
		byteCtr:             0,
	}

	err = b.setRstPin(true)
	if err == nil {
		err = b.setTestPin(true)
	}
	if err == nil {
		err = b.FlushInput()
	}
	if err == nil {
		err = b.FlushOutput()
	}

	return &b, err
}

// ComDone - Uzavřít spojení a resetovat FITkit
func (b *Instance) ComDone() error {
	err := b.setRstPin(false)
	if err == nil {
		err = b.setTestPin(false)
	}

	if err != nil {
		return err
	}
	return b.conn.Close()
}

func (b *Instance) Write(bytes []byte) (int, error) {
	n, err := b.conn.Write(bytes)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (b *Instance) Read(len int) ([]byte, error) {
	buff := make([]byte, len)

	received := 0
	for received < len {
		n, err := b.conn.Read(buff[received:])
		if err != nil {
			return nil, err
		}

		if n == 0 {
			return nil, errors.New(ErrReadTimeout)
		}

		received += n
	}

	return buff, nil
}

// SetReadTimeout - Nastaví na otevřeném spojení timeout čtení na ms
func (b *Instance) SetReadTimeout(ms int) error {
	return b.conn.Reconfigure(
		serial.WithReadTimeout(ms),
	)
}

// FlushInput - Resetuje buffer výstupních dat
func (b *Instance) FlushInput() error {
	return b.conn.ResetInputBuffer()
}

// FlushOutput - Resetuje buffer vstupních dat
func (b *Instance) FlushOutput() error {
	return b.conn.ResetOutputBuffer()
}

// SetBaudRate - Nastaví novou přenosovou rychlost na sériové lince
func (b *Instance) SetBaudRate(baudrate int) error {
	return b.conn.Reconfigure(
		serial.WithBaudrate(baudrate),
	)
}

// Nastaví RST/NMI pin na požadovanou hodnotu
func (b *Instance) setRstPin(level bool) error {
	err := b.conn.SetDTR(level)

	sleepMs(10)

	return err
}

// Nastaví TEST pin na požadovanou hodnotu
func (b *Instance) setTestPin(level bool) error {
	err := b.conn.SetRTS(level)

	sleepMs(10)

	return err
}

// BslReset Nastaví BSL sekvenci na RST a TEST pinech
// invokeBsl nastavuje, zda se použíje kompletní sekvence nebo
// jen nastavení RST pinu
func (b *Instance) BslReset(invokeBsl bool) error {
	err := b.setRstPin(true)
	if err == nil {
		err = b.setTestPin(true)
	}

	sleepMs(250)

	if err == nil {
		err = b.setRstPin(false)
	}

	if invokeBsl {
		if err == nil {
			err = b.setTestPin(true)
		}
		if err == nil {
			err = b.setTestPin(false)
		}
		if err == nil {
			err = b.setTestPin(true)
		}
		if err == nil {
			err = b.setTestPin(false)
		}
		if err == nil {
			err = b.setRstPin(true)
		}
		if err == nil {
			err = b.setTestPin(true)
		}
	} else {
		if err == nil {
			err = b.setRstPin(true)
		}
	}

	sleepMs(250)

	if err == nil {
		return err
	}
	return b.FlushInput()
}

// BslSync Odešle synchronizační znak a očekává potvrzující znak.
// Pokud je wait true, je opakováno dokud se to nepodaří (do nekonečna)
func (b *Instance) BslSync(wait bool) error {
	for loopcnt := 3; wait || loopcnt >= 0; loopcnt-- {
		err := b.FlushInput()
		var bytes []byte

		if err == nil {
			_, err = b.Write([]byte{constBslSync})
		}

		if err == nil {
			bytes, err = b.Read(1)
		}

		if err != nil {
			return err
		}

		if len(bytes) >= 1 && bytes[0] == constDataAck {
			// Synchronizace úspěšná
			return nil
		}
	}

	return errors.New(ErrBslSync)
}

// BslTxRx Odešle příkaz cmd do bootloaderu a vrátí odpověď (nebo nil)
// addr - počáteční adresa
// length - délka
// blkout - další data
// wait - viz bslSync()
func (b *Instance) BslTxRx(cmd byte, addr uint16, length uint16, blkout []byte, wait bool) ([]byte, error) {
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
	binary.LittleEndian.PutUint16(dataOut[0:], addr)
	binary.LittleEndian.PutUint16(dataOut[2:], length)

	if blkout != nil {
		dataOut = append(dataOut, blkout...)
	}

	err := b.BslSync(wait)
	if err != nil {
		return nil, err
	}

	rxFrame, err := b.ComTxRx(cmd, dataOut, uint16(len(dataOut)))
	if err != nil {
		return nil, err
	}

	if rxFrame != nil {
		return rxFrame[4:], nil // Pouze data bez [hdr,null,len,len]
	}

	return nil, nil
}

func (b *Instance) comRxHeader() (byte, byte, error) {
	hdr, err := b.Read(1)

	if err != nil {
		return 0, 0, nil
	}

	if len(hdr) < 1 {
		return 0, 0, errors.New("Rx header timeout")
	}

	rxHeader := hdr[0] & 0xf0
	b.reqNo = 0
	b.seqNo = 0
	rxNum := byte(0)

	return rxHeader, rxNum, nil
}

func (b *Instance) comRxFrame(rxNum byte) ([]byte, error) {
	rxFrame := make([]byte, 1)
	rxFrame[0] = constDataFrame | rxNum

	rxFrameData, err := b.Read(3)
	if err != nil {
		return nil, err
	}

	if len(rxFrameData) < 3 {
		return nil, errors.New("Rx frame timeout")
	}

	rxFrame = append(rxFrame, rxFrameData...)

	if rxFrame[1] != 0 || rxFrame[2] != rxFrame[3] {
		return nil, errors.New("Header corrupt")
	}

	rxLengthCRC := rxFrame[2] + 2
	rxFrameData, err = b.Read(int(rxLengthCRC))
	if err != nil {
		return nil, err
	}

	rxFrame = append(rxFrame, rxFrameData...)

	// rxLength+4: Length with header but w/o CRC:
	checksum := calcChecksum(rxFrame, uint16(rxFrame[2]+4))

	if rxFrame[rxFrame[2]+4] != byte(0xFF&checksum) || rxFrame[rxFrame[2]+5] != byte(0xff&(checksum>>8)) {
		return nil, errors.New("Checksum wrong " + ErrCom)
	}

	// Checksum correct, frame received correctly (=> send next frame)
	return rxFrame, nil
}

// ComTxRx Sends the command cmd with the data given in dataOut to the
// microcontroller and expects either an acknowledge or a frame
// with result from the microcontroller.  The results are stored
// in dataIn (if not a NULL pointer is passed).
// In this routine all the necessary protocol stuff is handled.
// Returns zero if the function was successful.
func (b *Instance) ComTxRx(cmd byte, dataOut []byte, length uint16) ([]byte, error) {
	if (length % 2) != 0 {
		dataOut = append(dataOut, 0xFF)
		length++
	}

	txFrame := make([]byte, 4)
	txFrame[0] = constDataFrame | b.seqNo
	txFrame[1] = cmd
	txFrame[2] = byte(len(dataOut))
	txFrame[3] = byte(len(dataOut))

	b.reqNo = (b.seqNo + 1) % constMaxFrameCount

	txFrame = append(txFrame, dataOut...)
	checksum := calcChecksum(txFrame, length+4)
	txFrame = append(txFrame, byte(checksum&0xFF), byte((checksum>>8)&0xFF))

	accessAddr := uint16(0x0212+(checksum^0xffff)) & 0xfffe // 0x0212: Address of wCHKSUM

	if b.bslMemAccessWarning && accessAddr < constBslCriticalAddr {
		log.Printf("WARNING: This command might change data at address %04x or %04x!\n", accessAddr, accessAddr+1)
	}

	err := b.FlushInput()
	if err != nil {
		return nil, err
	}

	_, err = b.Write(txFrame)
	if err != nil {
		return nil, err
	}

	rxHeader, rxNum, err := b.comRxHeader()
	if err != nil {
		return nil, err
	}

	switch rxHeader {
	case constDataAck: // Acknowledge/OK
		if rxNum == b.reqNo { // Acknowledge received correctly => next frame
			b.seqNo = b.reqNo
			return nil, nil
		}
		return nil, errors.New(ErrFrameNumber)
	case constDataNak: // Not acknowledge/error
		return nil, errors.New(ErrRxNAK)
	case constDataFrame: // Receive data
		if rxNum == b.reqNo {
			return b.comRxFrame(rxNum)
		}
		return nil, errors.New(ErrFrameNumber)
	case constCmdFailed: // Frame ok, but command failed.
		return nil, errors.New(ErrCmdFailed)
	}

	return nil, errors.New("Unknown header " + string(rxHeader) + "\nAre you downloading to RAM into an old device that requires the patch?")
}
