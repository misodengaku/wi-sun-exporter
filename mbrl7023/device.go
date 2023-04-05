package mbrl7023

import (
	"context"
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

type ChannelInfo struct {
	MACAddress  string `json:"mac_address"`
	IPv6Address string `json:"ipv6_address"`
	Channel     uint8  `json:"channel"`
	ChannelPage uint8  `json:"channel_page"`
	PairID      uint32 `json:"pair_id"`
	PanID       uint16 `json:"pan_id"`
	Side        uint32 `json:"side"` // wakaran
	LQI         uint8  `json:"lqi"`
}

var getInstantPowerBytes = []byte{0x10, 0x81, 0x00, 0x01, 0x05, 0xFF, 0x01, 0x02, 0x88, 0x01, 0x62, 0x01, 0xE7, 0x00}

func (m *MBRL7023) Init(ctx context.Context, devicePath string) error {
	port, err := serial.Open(devicePath, &serial.Mode{
		BaudRate: 115200,
	})
	if err != nil {
		return err
	}
	m.port = port

	// go func() {
	// 	var lineBuffer string

	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		default:
	// 			buffer := make([]byte, 128)
	// 			n, err := m.port.Read(buffer)
	// 			if err != nil {
	// 				panic(err)
	// 			}
	// 			if n == 0 {
	// 				continue
	// 			}
	// 			lineBuffer += strings.Trim(string(buffer), "\x00")

	// 			lines := strings.Split(lineBuffer, "\r\n")
	// 			if len(lines) == 1 {
	// 				continue
	// 			}
	// 			// for _, line := range lines[:len(lines)-1] {
	// 			// 	line = strings.TrimSuffix(line, "\r\n")
	// 			// 	line = strings.TrimPrefix(line, "OK ")
	// 			// 	u.parseLine(line)
	// 			// }
	// 			lineBuffer = lines[len(lines)-1]
	// 		}
	// 	}
	// }()
	return nil
}

func (m *MBRL7023) readLine(lineBuffer string) (line, remain string) {
	lines := strings.SplitN(lineBuffer, "\r\n", 2)
	if len(lines) == 2 {
		line = lines[0]
		remain = lines[1]
		return
	}
	for {
		buffer := make([]byte, 256)
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
	m.port.Write([]byte(fmt.Sprintf("SKSETPWD C %s\r\n", password)))
	for {
		line, remain = m.readLine(remain)
		println(line)
		if line == "OK" {
			break
		} else if strings.HasPrefix(line, "FAIL ") {
			return fmt.Errorf(line)
		}
	}

	remain = ""

	m.port.Write([]byte(fmt.Sprintf("SKSETRBID %s\r\n", id)))
	for {
		line, remain = m.readLine(remain)
		println(line)
		if line == "OK" {
			break
		} else if strings.HasPrefix(line, "FAIL ") {
			return fmt.Errorf(line)
		}
	}
	return nil
}

func (m *MBRL7023) parseLine(line string) error {
	// var err error
	elems := strings.Split(line, ",")
	for _, elem := range elems {
		kv := strings.Split(elem, "=")
		switch kv[0] {
		case "ID":
			m.ID = kv[1]
		case "VER":
			m.Version = kv[1]
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

func (m *MBRL7023) ChannelScan(scanDurationSec int) (ChannelInfo, error) {
	var line, remain string
	result := map[string]string{}
	m.port.Write([]byte(fmt.Sprintf("SKSCAN 2 FFFFFFFF %d 0\r\n", scanDurationSec)))
	for {
		line, remain = m.readLine(remain)
		println("line:", line, "remain: ", remain)
		if strings.HasPrefix(line, "EVENT 22") {
			break
		} else if strings.HasPrefix(line, "  ") {
			r := strings.SplitN(line, ":", 2)
			result[strings.TrimSpace(r[0])] = r[1]
		}
	}

	fmt.Printf("%#v\n", result)
	chInfo := ChannelInfo{}
	chInfo.MACAddress = result["Addr"]
	chInfo.IPv6Address = m.GetIPv6LinkLocalAddr(result["Addr"])
	v, err := strconv.ParseUint(result["Channel"], 16, 8)
	if err != nil {
		return ChannelInfo{}, err
	}
	chInfo.Channel = uint8(v)
	v, err = strconv.ParseUint(result["Channel Page"], 16, 8)
	if err != nil {
		return ChannelInfo{}, err
	}
	chInfo.ChannelPage = uint8(v)
	v, err = strconv.ParseUint(result["Pan ID"], 16, 16)
	if err != nil {
		return ChannelInfo{}, err
	}
	chInfo.PanID = uint16(v)
	v, err = strconv.ParseUint(result["PairID"], 16, 32)
	if err != nil {
		return ChannelInfo{}, err
	}
	chInfo.PairID = uint32(v)
	v, err = strconv.ParseUint(result["Side"], 16, 32)
	if err != nil {
		return ChannelInfo{}, err
	}
	chInfo.Side = uint32(v)
	v, err = strconv.ParseUint(result["LQI"], 16, 8)
	if err != nil {
		return ChannelInfo{}, err
	}
	chInfo.LQI = uint8(v)
	return chInfo, nil
}

func (m *MBRL7023) GetIPv6LinkLocalAddr(macAddr string) string {
	var line, remain string
	// result := map[string]string{}
	m.port.Write([]byte(fmt.Sprintf("SKLL64 %s\r\n", macAddr)))
	for {
		line, remain = m.readLine(remain)
		if !strings.HasPrefix(line, "SKLL64") {
			return line
		}
	}
}

func (m *MBRL7023) SetChannel(channel uint8) error {
	var line, remain string
	// result := map[string]string{}
	m.port.Write([]byte(fmt.Sprintf("SKSREG S2 %02X\r\n", channel)))
	for {
		line, remain = m.readLine(remain)
		if strings.HasPrefix(line, "OK") {
			return nil
		}
	}
}

func (m *MBRL7023) SetPanID(panID uint16) error {
	var line, remain string
	// result := map[string]string{}
	m.port.Write([]byte(fmt.Sprintf("SKSREG S3 %04X\r\n", panID)))
	for {
		line, remain = m.readLine(remain)
		if strings.HasPrefix(line, "OK") {
			return nil
		}
	}
}

func (m *MBRL7023) ExecutePANAAuth(ipv6Address string) error {
	var line, remain string
	m.port.Write([]byte(fmt.Sprintf("SKJOIN %s\r\n", ipv6Address)))
	for {
		line, remain = m.readLine(remain)
		if strings.HasPrefix(line, "OK") {
			return nil
		}
	}
}

func (m *MBRL7023) WaitForPANAAuth() error {
	var line, remain string
	for {
		line, remain = m.readLine(remain)
		if strings.HasPrefix(line, "EVENT 24") {
			return fmt.Errorf("failed to authentication")
		} else if strings.HasPrefix(line, "EVENT 25") {
			// success
			return nil
		}
	}
}

func (m *MBRL7023) GetInstantPower(ipv6Addr string) (uint32, error) {
	var line, remain string
	m.port.Write([]byte(fmt.Sprintf("SKSENDTO 1 %s 0E1A 1 0 %04X ", ipv6Addr, len(getInstantPowerBytes))))
	m.port.Write(getInstantPowerBytes)
	// m.readLine("") // skip echoback
	for {
		println("wait response")
		line, remain = m.readLine(remain)
		println(line)
		if strings.Contains(line, "ERXUDP") {
			// println("erxudp")
			// rx := strings.Split(line, "ERXUDP")
			elements := strings.Split(strings.TrimSpace(line), " ")
			println(elements)
			if len(elements) < 10 {
				continue
			}
			udpBody := elements[9]
			if len(udpBody) < 36 {
				continue
			}
			seoj := udpBody[8 : 8+6]
			println("seoj: ", seoj)
			esv := udpBody[20 : 20+2]
			println("esv: ", esv)
			if seoj == "028801" && esv == "72" {
				epc := udpBody[24 : 24+2]
				println("epc: ", epc)
				if epc == "E7" {
					power, err := strconv.ParseUint(udpBody[len(udpBody)-8:], 16, 32)
					if err != nil {
						return 0, err
					}
					return uint32(power), nil
				}
			}
		}
	}
	for {
		line, remain = m.readLine(remain)
		if strings.HasPrefix("ERXUDP", line) {
			elements := strings.Split(strings.TrimSpace(line), " ")
			udpBody := elements[9]
			seoj := udpBody[8 : 8+6]
			esv := udpBody[20 : 20+2]
			if seoj == "028801" && esv == "72" {
				power, err := strconv.ParseUint(udpBody[len(udpBody)-8-1:], 16, 32)
				if err != nil {
					return 0, err
				}
				return uint32(power), nil
			}
		}
	}

}
