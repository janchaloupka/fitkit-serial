// +build !darwin
// Knihovna albenik/go-serial neumožňuje cross platform sestavování enumeratoru
// pro cíl macOS, proto je speciální zdrojový kód discover_darwin.go bez použití
// této funkcionality. Jediný rozdíl ve funkčnosti, je že vyhledávání portů bude
// trvat trochu déle, protože se budou procházet i porty, které nedopovídají
// konrétnímu USB PID a VID

package discover

import (
	"errors"

	"github.com/albenik/go-serial/v2/enumerator"
)

// AllDevices - Najde všechny připojené FITkit přípravky a vrátí seznam těchto přípravků
func AllDevices() []Fitkit {
	found := []Fitkit{}

	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return found
	}

	for _, port := range ports {
		if port.IsUSB && port.VID == "0403" && port.PID == "6010" {
			info, err := GetFitkitInfo(port.Name)
			if err == nil {
				found = append(found, *info)
			}
		}
	}

	return found
}

// FirstDevice - Získá informaci o prvním nalezeném fitkitu
func FirstDevice() (*Fitkit, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}

	for _, port := range ports {
		if port.IsUSB && port.VID == "0403" && port.PID == "6010" {
			info, err := GetFitkitInfo(port.Name)
			if err == nil {
				return info, nil
			}
		}
	}

	return nil, errors.New("No available FITkit devices found")
}
