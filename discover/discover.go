package discover

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/tarm/serial"
	"go.bug.st/serial.v1/enumerator"
)

// Fitkit Struktura obsahující informace o nalezeném FITkitu
type Fitkit struct {
	Port     string // Sériový port, na kterém se FITkit nachází
	Version  string // Verze FITkitu (většinou 1.x nebo 2.x)
	Revision string // Číslo revize FITkitu
}

// Kontrola, zda se na sériovém portu nachází FITkit
func parseFitkitInfo(portName string, info *Fitkit) bool {
	fitkitRegex, _ := regexp.Compile("FITkit (.+) \\$Rev: (\\d+) \\$")

	config := serial.Config{
		Name:        portName,
		Baud:        460800,
		ReadTimeout: time.Millisecond * 500,
	}

	port, err := serial.OpenPort(&config)

	if err != nil {
		log.Fatal(err)
	}

	buff := make([]byte, 64)
	received := 0
	valid := false

	for {
		n, err := port.Read(buff[received:])
		if err != nil {
			log.Fatal(err)
			break
		}

		received += n

		submatch := fitkitRegex.FindSubmatch(buff)
		if len(submatch) >= 3 {
			info.Port = portName
			info.Version = string(submatch[1])
			info.Revision = string(submatch[2])
			valid = true
			break
		}

		if n == 0 || received >= len(buff) {
			break
		}
	}

	port.Close()
	return valid
}

// AllDevices Najde všechny připojené FITkit přípravky a vrátí seznam těchto přípravků
func AllDevices() []Fitkit {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		log.Fatal(err)
	}

	found := make([]Fitkit, 0)

	for _, port := range ports {
		if port.IsUSB && port.VID == "0403" && port.PID == "6010" {
			info := Fitkit{}
			if parseFitkitInfo(port.Name, &info) {
				found = append(found, info)
			}
		}
	}

	return found
}

// FirstDevice Získá informaci o prvním nalezeném fitkitu
func FirstDevice() Fitkit {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		log.Fatal(err)
	}

	for _, port := range ports {
		if port.IsUSB && port.VID == "0403" && port.PID == "6010" {
			info := Fitkit{}
			if parseFitkitInfo(port.Name, &info) {
				return info
			}
		}
	}

	log.Fatalln("No available FITkit devices found.")
	return Fitkit{}
}

// PrintDevices Vypíše na standardní výstup v JSON formátu seznam všech
// nalezených FITkit přípravků
func PrintDevices() {
	found := AllDevices()

	b, err := json.MarshalIndent(found, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))
}
