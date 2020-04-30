package fitkitbsl

import (
	"crypto/md5"
	"errors"
	"io"
	"os"
	"sort"

	"github.com/janch32/fitkit-relay/memory"
)

// Block pro nahrání do MCU
type Block struct {
	Address int
	Data    []byte
}

// IntelHexConvertToBlocks - Prekonvertuje segmenty tak, aby vznikly segmenty
// velikosti BLOCKSIZE bytu pokud nasleduje vice bloku bezprostredne zasebou,
// spoji je az dosahnou velikosti MAXBLOCKSIZE
func IntelHexConvertToBlocks(mem *memory.Memory, blockSize int, maxBlockSize int) ([]Block, error) {
	if maxBlockSize%blockSize > 0 {
		return nil, errors.New("maxBlockSize musi byt nasobek blockSize")
	}

	usedBlocks := []int{}
	usedBlocksLen := []int{}

	segments := mem.GetDataSegments()

	// Enumerace alokovanych bloku zarovnanych na BLOCKSIZE
	for _, s := range segments {
		sAddr := int(s.Address)
		lenSeg := len(s.Data)
		bAddr, bCnt := blockBounds(sAddr, lenSeg, blockSize)

		for i := bAddr; i < bAddr+bCnt*blockSize; i++ {
			if !inArray(usedBlocks, int(i)) {
				usedBlocks = append(usedBlocks, int(i))
				usedBlocksLen = append(usedBlocksLen, 1)
			}
		}

	}

	// Serazeni pouzitych bloku dle adresy a spojovani sousednich bloku
	sort.Ints(usedBlocks)

	lb := len(usedBlocks)
	maxLen := maxBlockSize / blockSize
	for b1 := 0; (b1 < lb) && (maxLen > 1); b1++ {
		if usedBlocksLen[b1] <= 0 {
			continue
		}

		bAddr := usedBlocks[b1]
		for b2 := b1 + 1; b2 < lb; b2++ {
			if usedBlocksLen[b2] > 0 {
				bMax := bAddr + usedBlocksLen[b1]*int(blockSize)
				if (usedBlocks[b2] == bMax) && usedBlocksLen[b1] < maxLen {
					usedBlocksLen[b2] = 0
					usedBlocksLen[b1]++
				} else {
					break
				}
			}
		}
	}

	// Vytvoreni novych segmentu a jejich naplneni daty
	blocks := []Block{}
	for i := 0; i < len(usedBlocks); i++ {
		bAddr := usedBlocks[i]
		bLen := usedBlocksLen[i]
		if bLen == 0 {
			continue
		}

		bMaxAddr := bAddr + bLen*blockSize - 1
		var bData []byte = nil
		for _, s := range segments {
			sAddr := int(s.Address)
			sData := s.Data
			lenSeg := len(sData)
			sMaxAddr := sAddr + lenSeg - 1

			if sAddr > bMaxAddr {
				break
			}

			if sMaxAddr < bAddr {
				continue
			}

			// segment zasahuje do bloku [baddr - bmaxaddr]
			sofs := max(0, bAddr-sAddr)
			bofs := max(0, sAddr-bAddr)
			seofs := min(bMaxAddr, sMaxAddr) - sAddr
			beofs := bofs + (seofs - sofs)

			if bData == nil {
				bData = make([]byte, bLen*blockSize)
				for j := 0; j < len(bData); j++ {
					bData[j] = 0xFF
				}
			}
			copy(bData[bofs:beofs+1], sData[sofs:seofs+1])
		}

		blocks = append(blocks, Block{
			Address: bAddr,
			Data:    bData,
		})

	}

	return blocks, nil
}

func blockBounds(addr int, len int, blockSize int) (int, int) {
	am := addr % blockSize
	if am > 0 {
		len += am
	}

	lm := len % blockSize
	ld := len / blockSize
	if lm > 0 {
		ld++
	}

	return (addr / blockSize) * blockSize, ld
}

func inArray(array []int, element int) bool {
	for _, e := range array {
		if e == element {
			return true
		}
	}

	return false
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Golang function used to generate MD5 digitest from file
// source: https://mrwaggel.be/post/generate-md5-hash-of-a-file-in-golang/
func hashFileMD5(file *os.File) ([]byte, error) {
	//Tell the program to call the following function when the current function returns
	defer file.Close()

	//Open a new hash interface to write to
	hash := md5.New()

	//Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	//Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]

	return hashInBytes, nil

}
