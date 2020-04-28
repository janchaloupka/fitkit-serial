package flash

import (
	"./bsl"
)

// Flash Spustí programování FITkitu na daném portu port
//
// Je nutné uvést cestu k binárnímu soubory pro programování FPGA
// a hex souboru pro naprogramování MCU. Pro MCU je nutné uvést minimálně
// jeden soubor (musí odpovídat připojenímu FITkitu).
// Lepší je uvést obě varianty MCU
func Flash(port string, mcuV1path string, mcuV2path string, fpgaPath string) {
	conn := bsl.Open(port)

	bsl.BslReset(conn, true)
	bsl.BslSync(conn, false)

	bsl.Close(conn)
}
