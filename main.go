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

// Computer хранит данные по компьютеру: имя, MAC-адрес и IP (с портом)
type Computer struct {
	Name string
	MAC  string
	IP   string // пример: "192.168.4.99:9"
}

// buildMagicPacket формирует magic packet по стандарту WOL
func buildMagicPacket(mac string) ([]byte, error) {
	mac = strings.ToLower(strings.ReplaceAll(mac, ":", ""))
	macBytes, err := hex.DecodeString(mac)
	if err != nil || len(macBytes) != 6 {
		return nil, fmt.Errorf("неправильный MAC-адрес: %s", mac)
	}

	// Пакет начинается с 6 байт 0xFF, затем 16 повторений MAC-адреса
	packet := make([]byte, 6+16*6)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], macBytes)
	}
	return packet, nil
}

// sendMagicPacket отправляет magic packet по указанным параметрам
func sendMagicPacket(mac, targetIP string) error {
	packet, err := buildMagicPacket(mac)
	if err != nil {
		return err
	}

	conn, err := net.Dial("udp", targetIP)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	return err
}

func main() {
	// Инициализация приложения Fyne
	a := app.New()
	w := a.NewWindow("Wake-on-LAN")

	// Список компьютеров (можешь заменить данные на свои)
	computers := []Computer{
		{"BIG", "18:c0:4d:8a:38:ce", "192.168.4.55:9"},
		{"Debian", "1c:69:7a:65:2d:98", "192.168.4.99:9"},
		{"NAS", "94:de:80:db:c7:02", "192.168.4.40:9"},
	}

	// Извлекаем имена для выпадающего списка
	names := []string{}
	for _, comp := range computers {
		names = append(names, comp.Name)
	}

	// Создание выпадающего списка
	selectComp := widget.NewSelect(names, func(selected string) {
		// можно реализовать динамическое обновление информации при выборе
	})
	selectComp.PlaceHolder = "Выберите компьютер"

	// Метка статуса для отображения результата отправки
	statusLabel := widget.NewLabel("Статус: не отправлено")

	// Кнопка для пробуждения выбранного компьютера
	wakeButton := widget.NewButton("Пробудить", func() {
		selectedName := selectComp.Selected
		if selectedName == "" {
			statusLabel.SetText("Статус: выберите компьютер")
			return
		}

		var comp *Computer
		for _, c := range computers {
			if c.Name == selectedName {
				comp = &c
				break
			}
		}
		if comp == nil {
			statusLabel.SetText("Статус: компьютер не найден")
			return
		}

		err := sendMagicPacket(comp.MAC, comp.IP)
		if err != nil {
			statusLabel.SetText(fmt.Sprintf("Ошибка: %v", err))
		} else {
			statusLabel.SetText("Пакет отправлен успешно!")
		}
	})

	// Создаем интерфейс с вертикальной компоновкой
	content := container.NewVBox(
		widget.NewLabel("Выберите компьютер для пробуждения:"),
		selectComp,
		wakeButton,
		statusLabel,
	)
	w.SetContent(content)
	w.Resize(fyne.NewSize(300, 200))
	w.ShowAndRun()
}
