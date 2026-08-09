<div align="center">

<img src="client/launcher/frontend/src/assets/images/appicon.png" width="120" alt="Minecraft Online Launcher">

# Minecraft Online Launcher

**Лаунчер сервера FamMC. Ставит Java, качает моды, обновляется сам.**
Просто скачайте, войдите и нажмите «Играть».

[![Релиз](https://img.shields.io/github/v/release/MirFlames/launcher?style=flat-square&label=версия&color=4c9f70)](https://github.com/MirFlames/launcher/releases/latest)
[![Загрузки](https://img.shields.io/github/downloads/MirFlames/launcher/total?style=flat-square&label=загрузок&color=4c9f70)](https://github.com/MirFlames/launcher/releases)
[![Telegram](https://img.shields.io/badge/Telegram-mc__fam-2AABEE?style=flat-square&logo=telegram&logoColor=white)](https://t.me/mc_fam)

</div>

---

## Скачать

<div align="center">

[![Windows](https://img.shields.io/badge/Windows-Скачать_установщик-0078D4?style=for-the-badge&logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0id2hpdGUiIGQ9Ik0wIDMuNUw5Ljc1IDIuMnY5LjNIMHpNMTAuOSAyLjA1TDI0IDB2MTEuNUgxMC45ek0wIDEyLjVoOS43NXY5LjNMMCAyMC41ek0xMC45IDEyLjVIMjRWMjRsLTEzLjEtMi4wNXoiLz48L3N2Zz4K)](https://github.com/MirFlames/launcher/releases/latest/download/minecraft-online-setup.exe)
[![macOS](https://img.shields.io/badge/macOS-Скачать_.dmg-000000?style=for-the-badge&logo=apple&logoColor=white)](https://github.com/MirFlames/launcher/releases/latest/download/launcher-macos.dmg)
[![Linux](https://img.shields.io/badge/Linux-Скачать_.tar.gz-FCC624?style=for-the-badge&logo=linux&logoColor=black)](https://github.com/MirFlames/launcher/releases/latest/download/launcher-linux.tar.gz)

</div>

<details>
<summary><b>Портативные версии (без установки)</b></summary>

<br>

| Система | Файл |
|---------|------|
| Windows | [launcher-windows.zip](https://github.com/MirFlames/launcher/releases/latest/download/launcher-windows.zip) |
| macOS | [launcher-macos.zip](https://github.com/MirFlames/launcher/releases/latest/download/launcher-macos.zip) |
| Linux | [launcher-linux.zip](https://github.com/MirFlames/launcher/releases/latest/download/launcher-linux.zip) |

Портативную версию распаковывайте **в отдельную папку** — рядом с лаунчером появятся файлы игры.
Не распаковывайте прямо в «Загрузки».

</details>

---

## Как начать играть

### 🪟 Windows

1. Запустите скачанный `minecraft-online-setup.exe` и пройдите установку.
2. Откройте лаунчер с рабочего стола или из меню «Пуск».
3. Нажмите «Войти» — откроется браузер для подтверждения входа.
4. Вернитесь в лаунчер и нажмите **«Играть»**.

### 🍎 macOS

1. Откройте скачанный `launcher-macos.dmg` и перетащите лаунчер в **Программы**.
2. При первом запуске macOS скажет, что не может проверить разработчика — это нормально, приложение пока без платной подписи Apple.
   **Нажмите на лаунчер правой кнопкой → «Открыть» → «Открыть»**, и дальше он будет запускаться обычным двойным щелчком.
3. Нажмите «Войти», подтвердите вход в браузере и возвращайтесь.
4. Нажмите **«Играть»**.

> Если пункт «Открыть» не помогает, зайдите в **Системные настройки → Конфиденциальность и безопасность** и нажмите «Всё равно открыть» рядом с именем лаунчера.

### 🐧 Linux

1. Распакуйте архив и сделайте файл исполняемым:
   ```bash
   tar -xzf launcher-linux.tar.gz && chmod +x launcher
   ```
2. Установите библиотеку веб-движка, если её ещё нет:
   ```bash
   sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0
   ```
   На Ubuntu 22.04 и старее пакет называется `libwebkit2gtk-4.0-37`.
3. Запустите `./launcher`, войдите и нажмите **«Играть»**.

---

## Что лаунчер делает за вас

| | |
|---|---|
| ☕ **Ставит Java сам** | Скачивает нужную версию OpenJDK под вашу систему. Отдельно ставить ничего не нужно. |
| 🔄 **Обновляется сам** | Новая версия приходит автоматически, подпись проверяется криптографически. |
| 🧩 **Синхронизирует моды** | Сборка модов всегда совпадает с серверной: лишнее удаляется, недостающее докачивается. |
| 🚀 **Подключает к серверу** | Адрес сервера уже прописан — искать и вводить ничего не надо. |
| 📊 **Показывает новости** | Прямо в окне лаунчера. |

---

## Где лежат файлы

**Игра, моды и Java:**

| Система | Путь |
|---------|------|
| Windows | рядом с `launcher.exe` |
| macOS | `~/Library/Application Support/minecraft-online` |
| Linux | `~/.local/share/minecraft-online` |

**Настройки и логи:**

| Система | Путь |
|---------|------|
| Windows | `%APPDATA%\FamMCLauncher` |
| macOS | `~/Library/Application Support/FamMCLauncher` |
| Linux | `~/.config/FamMCLauncher` |

---

## Если что-то пошло не так

**Сначала загляните в лог.** Он лежит в папке настроек (см. таблицу выше), в подпапке `logs`, файл `latest.log` — там написано, на каком шаге всё сломалось.

<details>
<summary><b>Игра не запускается / вылетает на старте</b></summary>

<br>

Проверьте `logs/latest.log`. Чаще всего причина — нехватка памяти или повреждённые файлы игры. Попробуйте удалить папку с файлами игры (см. таблицу выше) и запустить лаунчер заново — он скачает всё с нуля.

</details>

<details>
<summary><b>macOS: «Не удаётся проверить разработчика»</b></summary>

<br>

Приложение не подписано платным сертификатом Apple. Нажмите на лаунчер правой кнопкой → «Открыть» → «Открыть». Если не помогло:

```bash
xattr -dr com.apple.quarantine /Applications/launcher.app
```

</details>

<details>
<summary><b>Linux: окно не открывается или пустое</b></summary>

<br>

Почти всегда не хватает WebKit2GTK. Установите:

```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0
```

На Ubuntu 22.04 и старее — `libwebkit2gtk-4.0-37`. На Fedora — `gtk3 webkit2gtk4.1`.

</details>

<details>
<summary><b>Не скачивается Java</b></summary>

<br>

Лаунчер берёт OpenJDK с `api.adoptium.net`. Если у вас блокируется доступ к этому адресу, скачивание не пройдёт — попробуйте включить VPN на время первого запуска. Дальше Java уже будет установлена локально и интернет для неё не понадобится.

</details>

<details>
<summary><b>Ничего не помогло</b></summary>

<br>

Напишите в [Telegram-канал](https://t.me/mc_fam) и приложите файл `logs/latest.log` — так проблему найдут быстрее всего.

</details>

---

<div align="center">

**[💬 Telegram-канал сервера](https://t.me/mc_fam)**

Хотите поучаствовать в разработке? — [CONTRIBUTING.md](CONTRIBUTING.md)

</div>
