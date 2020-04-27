package terminal

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"go.bug.st/serial.v1"
)

// Open Otevře terminál komunikující s FITkitem na daném portu
func Open(port string) {
	conn, err := serial.Open(port, &serial.Mode{
		BaudRate: 460800,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	})

	if err != nil {
		log.Fatal(err)
	}

	go readSerial(conn)
	writeSerial(conn)
}

// Číst data, které posílá fitkit a vypisovat tato data na standardní výstup
func readSerial(conn serial.Port) {
	buffer := make([]byte, 100)

	for {
		n, err := conn.Read(buffer)

		if err != nil {
			log.Fatal(err)
			break
		}

		if n == 0 {
			fmt.Println("\nEOF")
			break
		}

		fmt.Printf("%v", string(buffer[:n]))
	}
}

// Číst data ze standardního vstupu a poslat je FITkitu
func writeSerial(conn serial.Port) {
	buffer := make([]byte, 100)
	reader := bufio.NewReader(os.Stdin)

	for {
		n, err := reader.Read(buffer)

		if err != nil {
			fmt.Println("")

			if err != io.EOF {
				log.Fatal(err)
			}

			break
		}

		conn.Write(buffer[:n])
	}
}
