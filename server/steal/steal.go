package steal

import (
	"context"
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

	fmt.Printf("Подключаюсь к %s...\n", server)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	conn, err := minecraft.Dialer{
		TokenSource: tokenSrc,
		ClientData: login.ClientData{
			GameVersion:      "1.21.132",
			DeviceOS:         protocol.DeviceAndroid,
			DeviceModel:      "samsung SM-G991B",
			LanguageCode:     "ru_RU",
			DefaultInputMode: 2,
			CurrentInputMode: 2,
			UIProfile:        1,
		},
	}.DialContext(ctx, "raknet", server)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}
	defer conn.Close()

	fmt.Println("Подключено! Получаю паки...")

	packs := conn.ResourcePacks()
	total := len(packs)
	if total == 0 {
		fmt.Println("Паков нет на сервере.")
		return nil
	}

	fmt.Printf("Найдено паков: %d\n", total)

	outDir := "/storage/emulated/0/packs/" + sanitizeName(server)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания папки: %w", err)
	}

	saved := 0
	for i, rp := range packs {
		sizeMB := float64(rp.Len()) / 1024 / 1024
		fmt.Printf("[%d/%d] %s v%s (%.2f МБ)...\n", i+1, total, rp.UUID(), rp.Version(), sizeMB)

		data, err := downloadPackWithProgress(rp)
		if err != nil {
			fmt.Printf("  ✗ Ошибка скачивания: %v\n", err)
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
		saved++
		fmt.Printf("  ✓ Сохранён: %s\n", savePath)
	}

	fmt.Printf("\nГотово! Сохранено %d/%d паков в %s\n", saved, total, outDir)
	return nil
}

func downloadPackWithProgress(rp *resource.Pack) ([]byte, error) {
	if rp.DownloadURL() != "" {
		fmt.Printf("  Скачиваю с URL...\n")
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
	lastPrint := time.Now()

	for off < total {
		n, err := rp.ReadAt(buf[off:], int64(off))
		if err != nil {
			if err == io.EOF {
				break
			}
			// Если соединение оборвалось — возвращаем что успели скачать
			if off > 0 {
				fmt.Printf("\n  Соединение оборвалось, сохраняю что успело скачаться (%d байт)...\n", off)
				return buf[:off], nil
			}
			return nil, err
		}
		off += n

		if time.Since(lastPrint) >= time.Second {
			pct := float64(off) / float64(total) * 100
			mb := float64(off) / 1024 / 1024
			totalMb := float64(total) / 1024 / 1024
			fmt.Printf("  %.1f%% (%.2f / %.2f МБ)\r", pct, mb, totalMb)
			lastPrint = time.Now()
		}
	}

	fmt.Println()
	return buf[:off], nil
}

func sanitizeName(name string) string {
	for _, c := range []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", " "} {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}
