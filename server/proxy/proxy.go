package proxy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"golang.org/x/oauth2"

	"github.com/vaniakakakak-tech/packsteal-server/pack"
)

func Run(server string, tokenSrc oauth2.TokenSource) error {
	if !strings.Contains(server, ":") {
		server = server + ":19132"
	}

	fmt.Println("Запускаю прокси...")
	fmt.Println("Зайди в Minecraft и подключись к: 127.0.0.1:19132")
	fmt.Println("Паки будут перехвачены автоматически")
	fmt.Println()

	p, err := minecraft.NewForeignStatusProvider(server)
	if err != nil {
		return fmt.Errorf("ошибка статус провайдера: %w", err)
	}

	listener, err := minecraft.ListenConfig{
		StatusProvider: p,
	}.Listen("raknet", "0.0.0.0:19132")
	if err != nil {
		return fmt.Errorf("ошибка запуска прокси: %w", err)
	}
	defer listener.Close()

	fmt.Println("Прокси запущен!")

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go handleConn(conn.(*minecraft.Conn), server, tokenSrc)
	}
}

func handleConn(clientConn *minecraft.Conn, server string, tokenSrc oauth2.TokenSource) {
	defer clientConn.Close()

	fmt.Println("Игрок подключился! Соединяюсь с сервером...")

	serverConn, err := minecraft.Dialer{
		TokenSource: tokenSrc,
		ClientData: login.ClientData{
			GameVersion:  "1.21.132",
			DeviceOS:     protocol.DeviceIOS,
			DeviceModel:  "Xiaomi Redmi Note 10",
			LanguageCode: "ru_RU",
		},
	}.DialTimeout("raknet", server, 2*time.Minute)
	if err != nil {
		fmt.Println("Ошибка подключения к серверу:", err)
		return
	}
	defer serverConn.Close()

	packs := serverConn.ResourcePacks()
	if len(packs) > 0 {
		fmt.Printf("Найдено паков: %d — сохраняю...\n", len(packs))
		savePacks(server, packs)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = clientConn.StartGame(serverConn.GameData())
	}()
	_ = serverConn.DoSpawn()
	<-done

	fmt.Println("Проксирую трафик...")

	go func() {
		for {
			pk, err := serverConn.ReadPacket()
			if err != nil {
				return
			}
			if err := clientConn.WritePacket(pk); err != nil {
				return
			}
		}
	}()

	for {
		pk, err := clientConn.ReadPacket()
		if err != nil {
			return
		}
		if err := serverConn.WritePacket(pk); err != nil {
			return
		}
	}
}

func savePacks(server string, packs []*resource.Pack) {
	outDir := "packs/" + sanitizeName(server)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Println("Ошибка создания папки:", err)
		return
	}

	total := len(packs)
	for i, rp := range packs {
		sizeMB := float64(rp.Len()) / 1024 / 1024
		fmt.Printf("[%d/%d] %s (%.2f МБ)...\n", i+1, total, rp.UUID(), sizeMB)

		data, err := downloadPack(rp)
		if err != nil {
			fmt.Printf("  ✗ Ошибка: %v\n", err)
			continue
		}

		p, err := pack.LoadResourcePackFromBytes(data)
		if err != nil {
			fmt.Printf("  ✗ Ошибка загрузки: %v\n", err)
			continue
		}

		if rp.Encrypted() {
			if err := p.Decrypt([]byte(rp.ContentKey())); err != nil {
				fmt.Printf("  ✗ Ошибка расшифровки: %v\n", err)
				continue
			}
		}

		savePath := fmt.Sprintf("%s/%d_%s.zip", outDir, i+1, sanitizeName(rp.UUID().String()))
		if err := p.Save(savePath); err != nil {
			fmt.Printf("  ✗ Ошибка сохранения: %v\n", err)
			continue
		}
		fmt.Printf("  ✓ Сохранён: %s\n", savePath)
	}
}

func downloadPack(rp *resource.Pack) ([]byte, error) {
	if rp.DownloadURL() != "" {
		resp, err := http.Get(rp.DownloadURL())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	total := rp.Len()
	buf := make([]byte, total)
	off := 0
	for off < total {
		n, err := rp.ReadAt(buf[off:], int64(off))
		if err != nil {
			if err == io.EOF {
				break
			}
			if off > 0 {
				return buf[:off], nil
			}
			return nil, err
		}
		off += n
	}
	return buf[:off], nil
}

func sanitizeName(name string) string {
	for _, c := range []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", " "} {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}
