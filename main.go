package main

import (
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
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
	conn, err := net.Dial("udp", targetIP)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(packet)
	return err
}

func getStorageFile(app fyne.App) (fyne.URI, error) {
	root := app.Storage().RootURI()
	return storage.Child(root, "computers.gob")
}

func saveComputers(app fyne.App, comps []Computer) {
	fileURI, err := getStorageFile(app)
	if err != nil {
		fmt.Println("Ошибка пути:", err)
		return
	}
	writer, err := storage.Writer(fileURI)
	if err != nil {
		fmt.Println("Ошибка открытия файла:", err)
		return
	}
	defer writer.Close()

	enc := gob.NewEncoder(writer)
	err = enc.Encode(comps)
	if err != nil {
		fmt.Println("Ошибка кодирования:", err)
	}
}

func loadComputers(app fyne.App) []Computer {
	fileURI, err := getStorageFile(app)
	if err != nil {
		fmt.Println("Ошибка пути:", err)
		return nil
	}
	reader, err := storage.Reader(fileURI)
	if err != nil {
		return nil
	}
	defer reader.Close()

	var comps []Computer
	dec := gob.NewDecoder(reader)
	err = dec.Decode(&comps)
	if err != nil && err != io.EOF {
		fmt.Println("Ошибка декодирования:", err)
	}
	return comps
}

func main() {
	a := app.New()
	w := a.NewWindow("Wake-on-LAN")

	computers := loadComputers(a)

	var selectComp *widget.Select
	statusLabel := widget.NewLabel("Статус: не отправлено")

	updateSelect := func() {
		names := []string{}
		for _, c := range computers {
			names = append(names, c.Name)
		}
		selectComp.Options = names
		selectComp.Refresh()
	}

	selectComp = widget.NewSelect([]string{}, func(selected string) {})
	selectComp.PlaceHolder = "Выберите компьютер"
	updateSelect()

	wakeButton := widget.NewButton("Пробудить", func() {
		selectedName := selectComp.Selected
		if selectedName == "" {
			statusLabel.SetText("Статус: выберите компьютер")
			return
		}
		var comp *Computer
		for i, c := range computers {
			if c.Name == selectedName {
				comp = &computers[i]
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

	addButton := widget.NewButton("+", func() {
		nameEntry := widget.NewEntry()
		macEntry := widget.NewEntry()
		ipEntry := widget.NewEntry()

		items := []*widget.FormItem{
			widget.NewFormItem("Имя", nameEntry),
			widget.NewFormItem("MAC", macEntry),
			widget.NewFormItem("IP:port", ipEntry),
		}

		dialog.ShowForm("Добавить компьютер", "Сохранить", "Отмена", items, func(ok bool) {
			if ok {
				computers = append(computers, Computer{
					Name: nameEntry.Text,
					MAC:  macEntry.Text,
					IP:   ipEntry.Text,
				})
				saveComputers(a, computers)
				updateSelect()
			}
		}, w)
	})

	removeButton := widget.NewButton(" - ", func() {
		selectedName := selectComp.Selected
		if selectedName == "" {
			statusLabel.SetText("Статус: выберите компьютер для удаления")
			return
		}
		for i, c := range computers {
			if c.Name == selectedName {
				computers = append(computers[:i], computers[i+1:]...)
				saveComputers(a, computers)
				updateSelect()
				selectComp.ClearSelected()
				statusLabel.SetText("Удалено")
				return
			}
		}
	})

	w.SetContent(container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Выберите компьютер для пробуждения:"),
			addButton,
			removeButton,
		),
		selectComp,
		wakeButton,
		statusLabel,
	))

	w.Resize(fyne.NewSize(350, 250))
	w.ShowAndRun()
}
