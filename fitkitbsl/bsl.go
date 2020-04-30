package fitkitbsl

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/janch32/fitkit-serial/memory"
	"github.com/janch32/fitkit-serial/mspbsl"
)

const (
	// ErrorTimeout - Bsl timeout
	ErrorTimeout = "FKBSL: Timeout"

	// ErrorInitResponse - Initialization error, no response
	ErrorInitResponse = "INIT: No response"

	// ErrorInfoInvalidCmd - Invalid command
	ErrorInfoInvalidCmd = "INFO: Invalid command"

	// ErrorInfoChecksum - Invalid checksum
	ErrorInfoChecksum = "INFO: Invalid checksum"

	// ErrorProgramResponse - Invalid response
	ErrorProgramResponse = "PROG: Invalid response"

	// ErrorProgramFail - Blok se nepodarilo ani jednou prenest bez chyb
	ErrorProgramFail = "PROG: Communication failed"

	// ErrorFlashInit - FPGA: Nedockali jsme se smazani bloku v externi FLASH
	ErrorFlashInit = "FLASH: Init timeout"

	// ErrorFlashCommand - FPGA: Invalid command
	ErrorFlashCommand = "FLASH: Invalid command"
)

func calculateChecksum(data []byte) byte {
	sum := byte(0)
	for _, b := range data {
		sum += b
	}

	return sum ^ 0xFF
}

// FITkitBSL - FITkit Bootstrap Loader functions
type FITkitBSL struct {
	mspBsl *mspbsl.Instance
	info   []byte
}

// NewFITkitBsl - Initialize new instance of FITkitBSL
func NewFITkitBsl(bsl *mspbsl.Instance) *FITkitBSL {
	return &FITkitBSL{
		mspBsl: bsl,
		info:   nil,
	}
}

func (b *FITkitBSL) actionInitialize() error {
	defer b.mspBsl.SetReadTimeout(1000) // Vrátit původní hodnotu po návratu z funkce
	b.mspBsl.SetReadTimeout(100)        // Čekat max 100ms

	attempts := 10
	for attempts > 0 {
		attempts--
		_, err := b.mspBsl.Write([]byte{0xF0, 0x0F})
		if err != nil {
			return err
		}

		res, err := b.mspBsl.Read(1)
		if err != nil {
			if err.Error() != mspbsl.ErrReadTimeout {
				return err
			}
		} else if res[0] == 0xF0 {
			return nil
		} else {
			return errors.New(ErrorInitResponse)
		}
	}

	if attempts <= 0 {
		return errors.New(ErrorTimeout)
	}

	return nil
}

func (b *FITkitBSL) actionReadInfo() error {
	attempts := 5

	for attempts > 0 {
		attempts--

		cmd, err := b.mspBsl.Read(1)
		if err != nil {
			if err.Error() == mspbsl.ErrReadTimeout {
				continue
			}
			return err
		}

		if cmd[0] == 0xFA {
			info, err := b.mspBsl.Read(264)
			if err != nil {
				return err
			}

			b.info = info
			chksum := calculateChecksum(info)
			chksumVerify, err := b.mspBsl.Read(1)
			if err != nil {
				return err
			}

			if chksum != chksumVerify[0] {
				return errors.New(ErrorInfoChecksum)
			}

			return nil
		}
		return errors.New(ErrorInfoInvalidCmd)
	}

	if attempts <= 0 {
		return errors.New(ErrorTimeout)
	}

	return nil
}

func (b *FITkitBSL) actionWriteInfo(data []byte) error {
	if len(data) != 264 {
		return errors.New("Cannot write info, data must be exactly 264 bytes long")
	}

	_, err := b.mspBsl.Write([]byte{0xFC})
	if err != nil {
		return err
	}

	attempts := 5

	chksum := calculateChecksum(data)

	zeroes := true
	for _, b := range data {
		if b != 0xFF {
			zeroes = false
			break
		}
	}

	for attempts > 0 {
		attempts--

		cmd, err := b.mspBsl.Read(1)
		if err != nil {
			if err.Error() == mspbsl.ErrReadTimeout {
				continue
			}
			return err
		}

		if cmd[0] == 0xF4 || cmd[0] == 0xF5 {
			// Page initialized or checksum error, (re)send requested data

			if zeroes {
				_, err = b.mspBsl.Write([]byte{chksum, 0x00})
			} else {
				_, err = b.mspBsl.Write(append([]byte{chksum, 0x01}, data...))
			}

			if err != nil {
				return err
			}

			if cmd[0] == 0xF4 {
				attempts = 5 // Page initialized reset attempt counter
			}
		} else if cmd[0] == 0xF7 {
			// All the blocks were written
			return nil
		} else {
			return errors.New(ErrorFlashCommand)
		}
	}

	if attempts <= 0 {
		return errors.New(ErrorTimeout)
	}

	return nil
}

func (b *FITkitBSL) actionProgramHEX(hexPath string, mcuErased bool) error {
	blockSize := 64
	maxBlockSize := 2 * blockSize

	mem, err := memory.LoadHexFile(hexPath)
	if err != nil {
		return err
	}

	// Load Intel HEX and create 64 or 128B data blocks
	hexData, err := IntelHexConvertToBlocks(mem, blockSize, maxBlockSize)
	if err != nil {
		return err
	}

	attempts := 5
	block := 0
	for attempts > 0 && block < len(hexData) {

		dataBlock := hexData[block]
		addr := dataBlock.Address

		// Prepare content of data buffer
		data := make([]byte, maxBlockSize)
		for i := len(dataBlock.Data); i < len(data); i++ {
			data[i] = 0xFF
		}

		copy(data[:len(dataBlock.Data)], dataBlock.Data)
		chck := calculateChecksum(data)

		// Write both blocks?
		flagsWriteBoth := byte(0x00)
		if len(dataBlock.Data) == maxBlockSize {
			flagsWriteBoth = 0x01
		}

		// Send packet
		packetCmd := byte(0xFA) // Vymaze FLASH pred prvnim zapisem
		if mcuErased {
			packetCmd = 0xFB
		}

		packet := append([]byte{packetCmd, byte(addr >> 8), byte(addr&0xC0) | flagsWriteBoth, chck}, data...)

		_, err = b.mspBsl.Write(packet)
		if err != nil {
			return err
		}

		// Check the response
		tries := 5
		res := byte(0)
		for tries > 0 {
			tries--
			resp, err := b.mspBsl.Read(1)
			if err != nil {
				if err.Error() != mspbsl.ErrReadTimeout {
					return err
				}
			} else {
				res = resp[0]
				tries = 1
				break
			}
		}

		if tries <= 0 { // No response received
			return errors.New(ErrorTimeout)
		}

		if res == 0xF1 { // Packet was successfully processed

			attempts = 5 // Send next packet (if any)
			block++
		} else if res == 0xF2 { // Checksum error, try to send the packet again
			attempts--
		} else {
			return errors.New(ErrorProgramResponse)
		}
	}

	if attempts <= 0 {
		return errors.New(ErrorProgramFail)
	}

	return nil
}

func (b *FITkitBSL) actionProgramBIN(binPath string) error {
	file, err := os.Open(binPath)
	if err != nil {
		return err
	}

	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := stat.Size()

	if fileSize <= 208*264 {
		_, err = b.mspBsl.Write([]byte{0xF4})
	} else if fileSize <= 805*264 {
		_, err = b.mspBsl.Write([]byte{0xF5})
	} else {
		return errors.New("Invalid file (too large): " + binPath)
	}

	if err != nil {
		return err
	}

	attempts := 10
	var packet []byte = nil

	for attempts > 0 {
		attempts--

		cmd, err := b.mspBsl.Read(1)
		if err != nil {
			if err.Error() == mspbsl.ErrReadTimeout {
				continue
			} else {
				return err
			}
		}

		if cmd[0] == 0xF4 { //Send content of the next page
			data := make([]byte, 264)
			_, err := file.Read(data)
			if err != nil && err != io.EOF {
				return err
			}

			chck := calculateChecksum(data)
			zeroes := true
			for _, b := range data {
				if b != 0x00 {
					zeroes = false
					break
				}
			}

			if zeroes {
				// block is empty, it is not necessary to send data
				packet = []byte{chck, 0x00}
			} else {
				packet = append([]byte{chck, 0x01}, data...)
			}

			_, err = b.mspBsl.Write(packet)
			if err != nil {
				return err
			}

			attempts = 5
		} else if cmd[0] == 0xF5 { // Checksum error, try to send the packet again
			if packet != nil {
				_, err = b.mspBsl.Write(packet)
				if err != nil {
					return err
				}
			}
		} else if cmd[0] == 0xF6 { // All the blocks were written
			return nil
		} else {
			return errors.New(ErrorFlashCommand)
		}
	}

	return errors.New(ErrorFlashInit)
}

// Program loads provided BIN and HEX files to the FITkit device
func (b *FITkitBSL) Program(mcuHexPath string, fpgaBinPath string, mcuErased bool, force bool) error {
	// Load hex file info
	hexFile, err := os.Open(mcuHexPath)
	if err != nil {
		return err
	}

	defer hexFile.Close()
	hexFileInfo, err := hexFile.Stat()
	if err != nil {
		return err
	}

	hexFileModTime := uint32(hexFileInfo.ModTime().Unix())
	hexFileDigitest, err := hashFileMD5(hexFile)
	if err != nil {
		return err
	}

	// Load bin file info
	binFile, err := os.Open(fpgaBinPath)
	if err != nil {
		return err
	}

	defer hexFile.Close()
	binFileInfo, err := binFile.Stat()
	if err != nil {
		return err
	}

	binFileModTime := uint32(binFileInfo.ModTime().Unix())
	binFileDigitest, err := hashFileMD5(binFile)
	if err != nil {
		return err
	}

	fmt.Println("FITkit bootloader handshake...")
	err = b.actionInitialize()
	if err != nil {
		return err
	}

	fmt.Println("Reading info...")
	err = b.actionReadInfo()
	if err != nil {
		return err
	}

	_, err = NewInfo(b.info)
	if err != nil {
		return err
	}

	fkInfo, err := NewInfo(b.info)
	if err != nil {
		return err
	}

	hexInfoDigitest, _, hexInfoModTime, _, _, err := fkInfo.GetHexInfo()
	if err != nil {
		return err
	}

	binInfoDigitest, _, binInfoModTime, _, _, err := fkInfo.GetBinInfo()
	if err != nil {
		return err
	}

	hexEqual := (bytes.Compare(hexInfoDigitest, hexFileDigitest) == 0) && (hexInfoModTime == hexFileModTime)
	binEqual := (bytes.Compare(binInfoDigitest, binFileDigitest) == 0) && (binInfoModTime == binFileModTime)

	fmt.Println("")
	if !hexEqual || mcuErased || force {
		fmt.Println("Uploading MCU HEX data...")
		fmt.Println(mcuHexPath)

		err = b.actionProgramHEX(mcuHexPath, mcuErased)
		if err != nil {
			return err
		}

		fkInfo.UpdateHexInfo(hexFileDigitest, hexFileModTime, mcuHexPath)
	} else {
		fmt.Println("Skipping MCU flash, digitest matches (identical data are already in the MCU flash)")
	}

	fmt.Println("")
	if !binEqual || force {
		fmt.Println("Uploading FPGA binary data...")
		fmt.Println(fpgaBinPath)

		err = b.actionProgramBIN(fpgaBinPath)
		if err != nil {
			return err
		}

		fkInfo.UpdateBinInfo(binFileDigitest, binFileModTime, fpgaBinPath)
	} else {
		fmt.Println("Skipping FPGA flash, digitest matches (identical data are already in the FPGA flash)")
	}

	fmt.Println("")
	fmt.Println("Writing info...")
	err = b.actionWriteInfo(fkInfo.GetRawData())
	if err != nil {
		return err
	}

	// Send end handshake
	b.mspBsl.Write([]byte{0xFD})
	fmt.Println("Success")

	return nil
}
