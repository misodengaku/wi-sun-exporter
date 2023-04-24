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
var ErrTimeout = fmt.Errorf("timeout")

func (m *MBRL7023) Init(devicePath string) error {
	port, err := serial.Open(devicePath, &serial.Mode{
		BaudRate: 115200,
	})
	if err != nil {
		return err
	}
	m.port = port
	m.port.SetReadTimeout(250 * time.Millisecond)

	return nil
}

func (m *MBRL7023) readLine(ctx context.Context, lineBuffer string) (line, remain string, _err error) {
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
		select {
		case <-ctx.Done():
			_err = ErrTimeout
			return
		default:
			if n == 0 {
				continue
			}
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

func (m *MBRL7023) SetAuthentication(ctx context.Context, id, password string) error {
	var line, remain string
	var err error
	m.port.Write([]byte(fmt.Sprintf("SKSETPWD C %s\r\n", password)))
	for {
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return err
		}
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
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return err
		}
		println(line)
		if line == "OK" {
			break
		} else if strings.HasPrefix(line, "FAIL ") {
			return fmt.Errorf(line)
		}
	}
	return nil
}

func (m *MBRL7023) ChannelScan(ctx context.Context, scanDurationSec int) (ChannelInfo, error) {
	var line, remain string
	var err error
	result := map[string]string{}
	m.port.Write([]byte(fmt.Sprintf("SKSCAN 2 FFFFFFFF %d 0\r\n", scanDurationSec)))
	for {
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return ChannelInfo{}, err
		}
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
	chInfo.IPv6Address, err = m.GetIPv6LinkLocalAddr(ctx, result["Addr"])
	if err != nil {
		return ChannelInfo{}, err
	}
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

func (m *MBRL7023) GetIPv6LinkLocalAddr(ctx context.Context, macAddr string) (string, error) {
	var line, remain string
	var err error
	// result := map[string]string{}
	m.port.Write([]byte(fmt.Sprintf("SKLL64 %s\r\n", macAddr)))
	for {
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(line, "SKLL64") {
			return line, nil
		}
	}
}

func (m *MBRL7023) SetChannel(ctx context.Context, channel uint8) error {
	var line, remain string
	var err error
	// result := map[string]string{}
	m.port.Write([]byte(fmt.Sprintf("SKSREG S2 %02X\r\n", channel)))
	for {
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "OK") {
			return nil
		}
	}
}

func (m *MBRL7023) SetPanID(ctx context.Context, panID uint16) error {
	var line, remain string
	var err error
	// result := map[string]string{}
	m.port.Write([]byte(fmt.Sprintf("SKSREG S3 %04X\r\n", panID)))
	for {
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "OK") {
			return nil
		}
	}
}

func (m *MBRL7023) ExecutePANAAuth(ctx context.Context, ipv6Address string) error {
	var line, remain string
	var err error
	m.port.Write([]byte(fmt.Sprintf("SKJOIN %s\r\n", ipv6Address)))
	for {
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "OK") {
			return nil
		}
	}
}

func (m *MBRL7023) WaitForPANAAuth(ctx context.Context) error {
	var line, remain string
	var err error
	for {
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "EVENT 24") {
			return fmt.Errorf("failed to authentication")
		} else if strings.HasPrefix(line, "EVENT 25") {
			// success
			return nil
		}
	}
}

func (m *MBRL7023) GetInstantPower(ctx context.Context, ipv6Addr string) (uint32, error) {
	var line, remain string
	var err error
	m.port.Write([]byte(fmt.Sprintf("SKSENDTO 1 %s 0E1A 1 0 %04X ", ipv6Addr, len(getInstantPowerBytes))))
	m.port.Write(getInstantPowerBytes)
	// m.readLine("") // skip echoback
	for {
		println("wait response")
		line, remain, err = m.readLine(ctx, remain)
		if err != nil {
			return 0, err
		}
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
	// for {
	// 	line, remain, err = m.readLine(ctx, remain)
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// 	if strings.HasPrefix("ERXUDP", line) {
	// 		elements := strings.Split(strings.TrimSpace(line), " ")
	// 		udpBody := elements[9]
	// 		seoj := udpBody[8 : 8+6]
	// 		esv := udpBody[20 : 20+2]
	// 		if seoj == "028801" && esv == "72" {
	// 			power, err := strconv.ParseUint(udpBody[len(udpBody)-8-1:], 16, 32)
	// 			if err != nil {
	// 				return 0, err
	// 			}
	// 			return uint32(power), nil
	// 		}
	// 	}
	// }

}
