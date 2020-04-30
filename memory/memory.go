package memory

import (
	"os"
	"strconv"
	"strings"

	"github.com/marcinbor85/gohex"
)

// Memory - Reprezentuje obsah paměti
type Memory struct {
	*gohex.Memory
}

// GetMemRange - Získat data z pěměti v zadaném rozpětí. Nedefinované hodnoty jsou vyplněny 0xFF
func (m Memory) GetMemRange(fromAddr uint32, toAddr uint32) []byte {
	segments := m.GetDataSegments()

	res := make([]byte, 0)
	toAddr++

	for fromAddr < toAddr {
		found := false

		for _, seg := range segments {
			segEnd := seg.Address + uint32(len(seg.Data))

			if seg.Address <= fromAddr && fromAddr < segEnd {
				found = true
				catchLength := toAddr - fromAddr
				if toAddr > segEnd {
					// Všechna data nejsou v segmentu
					catchLength = segEnd - fromAddr
				}

				res = append(res, seg.Data[fromAddr-seg.Address:fromAddr-seg.Address+catchLength]...)
				fromAddr += catchLength

				if uint32(len(res)) >= toAddr-fromAddr {
					return res
				}
			}
		}

		if !found {
			res = append(res, []byte{0xFF}...)
			fromAddr++
		}
	}

	return res
}

// LoadHexFile - Načte binární obsah z .hex soubor (Intel HEX formát)
func LoadHexFile(path string) (*Memory, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	mem := gohex.NewMemory()
	err = mem.ParseIntelHex(file)
	if err != nil {
		return nil, err
	}

	return &Memory{mem}, nil
}

// LoadTIText - Načte data z řetězce ve formátu TI-Text
func LoadTIText(data string) (*Memory, error) {
	mem := gohex.NewMemory()

	startAddr := uint32(0)
	segmentData := []byte{}

	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == 'q' {
			break
		}

		if line[0] == '@' {
			if len(segmentData) > 0 {
				mem.AddBinary(startAddr, segmentData)
			}

			addr, err := strconv.ParseUint(line[1:], 16, 32)
			if err != nil {
				return nil, err
			}

			startAddr = uint32(addr)
			segmentData = []byte{}
		} else {
			for _, val := range strings.Fields(line) {
				b, err := strconv.ParseUint(val, 16, 8)
				if err != nil {
					return nil, err
				}

				segmentData = append(segmentData, byte(b))
			}
		}
	}

	if len(segmentData) > 0 {
		mem.AddBinary(startAddr, segmentData)
	}

	return &Memory{mem}, nil
}
