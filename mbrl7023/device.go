package mbrl7023

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
)

type MBRL7023 struct {
	port        serial.Port
	ID          string
	Version     string
	FRC         int
	Timestamp   time.Time
	CO2         int
	Humidity    float64
	Temperature float64
}

func (m *MBRL7023) Init(ctx context.Context, devicePath string) error {
	port, err := serial.Open(devicePath, &serial.Mode{
		BaudRate: 115200,
	})
	if err != nil {
		return err
	}
	m.port = port

	go func() {
		var lineBuffer string

		for {
			select {
			case <-ctx.Done():
				return
			default:
				buffer := make([]byte, 128)
				n, err := u.port.Read(buffer)
				if err != nil {
					panic(err)
				}
				if n == 0 {
					continue
				}
				lineBuffer += strings.Trim(string(buffer), "\x00")

				lines := strings.Split(lineBuffer, "\r\n")
				if len(lines) == 1 {
					continue
				}
				// for _, line := range lines[:len(lines)-1] {
				// 	line = strings.TrimSuffix(line, "\r\n")
				// 	line = strings.TrimPrefix(line, "OK ")
				// 	u.parseLine(line)
				// }
				lineBuffer = lines[len(lines)-1]
			}
		}
	}()
	return nil
}

func (m *MBRL7023) readLine(buffer string) (line, remain string) {
	var lineBuffer string
	lineBuffer += buffer
	for {

		buffer := make([]byte, 128)
		n, err := m.port.Read(buffer)
		if err != nil {
			panic(err)
		}
		if n == 0 {
			continue
		}

		lineBuffer += strings.Trim(string(buffer), "\x00")

		lines := strings.SplitN(lineBuffer, "\r\n", 2)
		if len(lines) == 1 {
			continue
		}
		line = lines[0]
		remain = lines[1]
		return
	}

}

func (m *MBRL7023) SetAuthentication(id, password string) error {
	var line, remain string
	for {
		m.port.Write([]byte(fmt.Sprintf("SKSETPWD C %s\r\n", password)))
		line, remain = m.readLine(remain)
		if line == "OK" {
			break
		}
	}

	for {
		m.port.Write([]byte(fmt.Sprintf("SKSETRBID %s\r\n", id)))
		line, remain = m.readLine(remain)
		if line == "OK" {
			break
		}
	}
	return nil
}

func (m *MBRL7023) parseLine(line string) error {
	var err error
	elems := strings.Split(line, ",")
	for _, elem := range elems {
		kv := strings.Split(elem, "=")
		switch kv[0] {
		case "CO2":
			u.CO2, err = strconv.Atoi(kv[1])
			if err != nil {
				return err
			}
			u.Timestamp = time.Now()
		case "HUM":
			u.Humidity, err = strconv.ParseFloat(kv[1], 64)
			if err != nil {
				return err
			}
			u.Timestamp = time.Now()
		case "TMP":
			u.Temperature, err = strconv.ParseFloat(kv[1], 64)
			if err != nil {
				return err
			}
			u.Timestamp = time.Now()
		case "ID":
			u.ID = kv[1]
		case "VER":
			u.Version = kv[1]
		}
	}
	return nil
}

func (m *MBRL7023) readResult() (result string, success bool) {
	buffer := make([]byte, 1024)
	result = strings.TrimRight(string(buffer), " \r\n")
	success = result != "NG"
	if success {
		result = strings.TrimPrefix(result, "OK ")
	}
	return
}

func (m *MBRL7023) GetDeviceID() string {
	u.port.Write([]byte("ID?\r\n"))
	result, ok := u.readResult()
	if !ok {
		panic(result)
	}
	return result
}

func (m *MBRL7023) GetFirmwareVersion() string {
	u.port.ResetInputBuffer()
	u.port.Write([]byte("VER?\r\n"))
	result, ok := u.readResult()
	if !ok {
		panic(result)
	}
	return result
}

func (m *MBRL7023) StartMeasurement() string {
	u.port.ResetInputBuffer()
	u.port.Write([]byte("STA\r\n"))
	result, ok := u.readResult()
	if !ok {
		panic(result)
	}
	return result
}

func (m *MBRL7023) StopMeasurement() {
	u.port.Write([]byte("STP\r\n"))
}

func (m *MBRL7023) GetFRCValue() {
	u.port.Write([]byte("FRC?\r\n"))
}

func (m *MBRL7023) setFRCValue(frcValue int) error {
	if frcValue < 400 || frcValue > 2000 {
		return errors.New("frcValue is out of range")
	}
	cmd := fmt.Sprintf("FRC=%d\r\n", frcValue)
	u.port.Write([]byte(cmd))
	return nil
}
