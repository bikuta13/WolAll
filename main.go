package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Computer struct {
	Name string
	MAC  string
	IP   string
}

func buildMagicPacket(mac string) ([]byte, error) {
	mac = strings.ToLower(strings.ReplaceAll(mac, ":", ""))
	macBytes, err := hex.DecodeString(mac)
	if err != nil || len(macBytes) != 6 {
		return nil, fmt.Errorf("неправильный MAC-адрес: %s", mac)
	}

	packet := make([]byte, 6+16*6)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], macBytes)
	}
	return packet, nil
}

func sendMagicPacket(mac, targetIP string) error {
	packet, err := buildMagicPacket(mac)
	if err != nil {
		return err
	}

	addr, err := net.ResolveUDPAddr("udp", targetIP)
	if err != nil {
		return fmt.Errorf("ошибка ResolveUDPAddr: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("ошибка DialUDP: %v", err)
	}
	defer conn.Close()

	fmt.Printf("Отправляю WOL на %s (MAC: %s)\n", targetIP, mac)

	_, err = conn.Write(packet)
	return err
}

func main() {
	a := app.New()
	w := a.NewWindow("Wake-on-LAN")

	// Устройства с широковещательным IP
	computers := map[string]Computer{
		"BIG":    {"BIG", "1c:69:7a:65:2d:98", "192.168.4.255:9"},
		"Debian": {"Debian", "a2:dd:6c:02:9b:a4", "192.168.4.255:9"},
		"NAS":    {"NAS", "94:de:80:db:c7:02", "192.168.4.255:9"},
	}

	statusLabel := widget.NewLabel("Статус: ожидание...")

	// Функция для создания кнопки
	makeButton := func(name string) *widget.Button {
		return widget.NewButton(name, func() {
			comp := computers[name]
			err := sendMagicPacket(comp.MAC, comp.IP)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Ошибка (%s): %v", name, err))
				fmt.Println("Ошибка:", err)
			} else {
				statusLabel.SetText(fmt.Sprintf("%s: пакет отправлен!", name))
				fmt.Println(name, "→ Magic Packet отправлен.")
			}
		})
	}

	// UI
	content := container.NewVBox(
		widget.NewLabel("Выберите устройство для пробуждения:"),
		makeButton("BIG"),
		makeButton("Debian"),
		makeButton("NAS"),
		statusLabel,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(300, 220))
	w.ShowAndRun()
}
