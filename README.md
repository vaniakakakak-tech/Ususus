# PackSteal

Приложение для скачивания ресурспаков с Minecraft Bedrock серверов.

## Сборка APK

1. Загрузи проект на GitHub
2. GitHub Actions автоматически соберёт APK
3. Скачай APK в разделе Actions → Artifacts

## Запуск в Termux

```bash
cd server
go mod tidy
CGO_ENABLED=0 go build -o packsteal .
./packsteal steal zeqa.net:19132
```
