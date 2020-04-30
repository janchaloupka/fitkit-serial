package discover

import (
	"errors"

	"github.com/albenik/go-serial/v2"
)

// AllDevices - Najde všechny připojené FITkit přípravky a vrátí seznam těchto přípravků
func AllDevices() []Fitkit {
	found := []Fitkit{}

	ports, err := serial.GetPortsList()
	if err != nil {
		return found
	}

	for _, port := range ports {
		info, err := GetFitkitInfo(port)
		if err == nil {
			found = append(found, *info)
		}
	}

	return found
}

// FirstDevice - Získá informaci o prvním nalezeném fitkitu
func FirstDevice() (*Fitkit, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}

	for _, port := range ports {
		info, err := GetFitkitInfo(port)
		if err == nil {
			return info, nil
		}
	}

	return nil, errors.New("No available FITkit devices found")
}
