# Backend сервис автообновления лаунчера

Легковесный HTTP сервер на Go для предоставления информации о версиях клиентских файлов и модов для Minecraft лаунчера.

## Требования

- Go 1.21 или выше

## Установка и запуск

1. Установите зависимости:
```bash
go mod tidy
```

2. Настройте конфигурацию в файле `config.json`:
   - `minecraft_version` - версия Minecraft
   - `mods_hash` - хеш модов (может быть пустым)
   - `client_files` - список клиентских файлов с их URL и SHA-256 хешами
   - `mods` - список модов с их URL и SHA-256 хешами
   - `files_path` - путь к директории с файлами для раздачи
   - `port` - порт сервера (по умолчанию 8080)
   - `jdk.download_url` - (опционально) альтернативный URL для скачивания JDK, если Adoptium недоступен (например, без VPN)

3. Поместите файлы в директорию, указанную в `files_path`:
   ```
   files/
   ├── versions/
   │   └── 1.20.1/
   │       └── client.jar
   └── mods/
       └── example-mod.jar
   ```

   **После добавления новых модов** в `files/mods` выполните пересчёт конфигурации:
   ```bash
   go run . -rescan
   ```
   Это обновит `config.json` списком модов и их хешами. Затем запустите сервер без флага `-rescan`.

4. Запустите сервер:
```bash
go run .
```

Или скомпилируйте и запустите:
```bash
go build -o launcher-backend
./launcher-backend
```

## API

### GET /api/version

Возвращает манифест версии с информацией о файлах клиента и модах.

**Ответ:**
```json
{
  "minecraft_version": "1.20.1",
  "mods_hash": "",
  "client_files": [
    {
      "name": "client.jar",
      "url": "http://localhost:8080/files/versions/1.20.1/client.jar",
      "hash": "sha256_хеш_в_hex_формате"
    }
  ],
  "mods": [
    {
      "name": "example-mod.jar",
      "url": "http://localhost:8080/files/mods/example-mod.jar",
      "hash": "sha256_хеш_в_hex_формате"
    }
  ]
}
```

### GET /files/{путь_к_файлу}

Раздает файлы для скачивания. Путь должен соответствовать структуре в `files_path`.

**Заголовки ответа:**
- `Content-Length` - размер файла (обязательно для прогресса загрузки)
- `Accept-Ranges: bytes` - поддержка докачки
- `Content-Type: application/octet-stream`

## Вычисление SHA-256 хеша

Для вычисления SHA-256 хеша файла используйте:

**Windows (PowerShell):**
```powershell
Get-FileHash -Path "путь\к\файлу.jar" -Algorithm SHA256 | Select-Object -ExpandProperty Hash | ForEach-Object { $_.ToLower() }
```

**Linux/Mac:**
```bash
sha256sum файл.jar | awk '{print $1}'
```

**Go:**
```go
package main

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"
)

func main() {
    file, _ := os.Open("файл.jar")
    defer file.Close()
    
    hash := sha256.New()
    io.Copy(hash, file)
    fmt.Println(hex.EncodeToString(hash.Sum(nil)))
}
```

## Безопасность

- Проверка на path traversal (запрет `..` и абсолютных путей)
- Валидация путей файлов
- Проверка существования файлов перед раздачей

## Производительность

Сервер оптимизирован для работы на слабом оборудовании:
- Минимальное потребление памяти
- Эффективная обработка запросов
- Быстрая раздача статических файлов
- Низкая задержка ответа

## Пример конфигурации

См. файл `config.json` для примера конфигурации.
