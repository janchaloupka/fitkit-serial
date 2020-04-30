package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/albenik/go-serial/v2"
)

// OpenTerminal Otevře terminál komunikující s FITkitem na daném portu
func OpenTerminal(port string) {
	conn, err := serial.Open(
		port,
		serial.WithBaudrate(460800),
		serial.WithDataBits(8),
		serial.WithStopBits(serial.OneStopBit),
		serial.WithParity(serial.NoParity),
		serial.WithHUPCL(true),
		serial.WithReadTimeout(100),
	)

	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	closed := false

	go readSerial(conn, &closed)
	writeSerial(conn)
	closed = true
}

// Číst data, které posílá FITkit a vypisovat tato data na standardní výstup
func readSerial(conn *serial.Port, isClosed *bool) {
	buffer := make([]byte, 100)

	for {
		n, err := conn.Read(buffer)

		if *isClosed {
			break
		}

		if err != nil {
			log.Fatal(err)
			break
		}

		fmt.Print(string(buffer[:n]))
	}
}

// Číst data ze standardního vstupu a poslat je FITkitu
func writeSerial(conn *serial.Port) {
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
