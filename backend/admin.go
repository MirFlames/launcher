package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

type adminSessionView struct {
	Nickname        string
	LauncherVersion string
	LastLoginHuman  string
}

func startAdminServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/sessions", handleAdminSessions)

	addr := ":8090"
	log.Printf("[Admin] Панель запущена на %s (не забудьте ограничить доступ на уровне сети)", addr)
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("[Admin] Ошибка ListenAndServe: %v", err)
		}
	}()
}

func handleAdminSessions(w http.ResponseWriter, _ *http.Request) {
	type entry struct {
		Nickname        string
		LauncherVersion string
		LastLoginHuman  string
	}

	raw := sessionGetAllForAdmin()
	var items []entry
	for _, e := range raw {
		last := "никогда"
		if e.LastLoginAt != nil && *e.LastLoginAt > 0 {
			last = time.Unix(*e.LastLoginAt, 0).Format("02.01.2006 15:04")
		}
		items = append(items, entry{
			Nickname:        e.Nickname,
			LauncherVersion: e.LauncherVersion,
			LastLoginHuman:  last,
		})
	}

	tmpl := `
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>Админ-панель — сессии</title>
  <style>
    body { font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background:#0b1020; color:#f5f5f5; margin:0; padding:24px; }
    h1 { margin-bottom: 16px; }
    table { border-collapse: collapse; width: 100%; max-width: 960px; background:#11172a; border-radius:8px; overflow:hidden; }
    th, td { padding: 8px 12px; border-bottom:1px solid rgba(255,255,255,0.06); text-align:left; font-size:14px; }
    th { background:#161d33; font-weight:600; }
    tr:nth-child(even) { background:#101626; }
    .tag { display:inline-flex; align-items:center; padding:2px 8px; border-radius:999px; font-size:12px; background:#1e293b; }
    .tag--legacy { background:#7c2d12; color:#fff; }
  </style>
</head>
<body>
  <h1>Сессии игроков</h1>
  <table>
    <thead>
      <tr>
        <th>Никнейм</th>
        <th>Версия лаунчера</th>
        <th>Последний вход</th>
      </tr>
    </thead>
    <tbody>
      {{- if not . }}
      <tr><td colspan="3">Нет данных о сессиях.</td></tr>
      {{- else }}
      {{- range . }}
      <tr>
        <td>{{ .Nickname }}</td>
        <td>
          {{ if .LauncherVersion }}
            <span class="tag{{ if (hasPrefix .LauncherVersion "0.") }} tag--legacy{{ end }}">{{ .LauncherVersion }}</span>
          {{ else }}
            <span class="tag tag--legacy">неизвестно</span>
          {{ end }}
        </td>
        <td>{{ .LastLoginHuman }}</td>
      </tr>
      {{- end }}
      {{- end }}
    </tbody>
  </table>
</body>
</html>`

	// Встроенная функция для пометки старых версий.
	funcs := template.FuncMap{
		"hasPrefix": func(s, prefix string) bool {
			return len(s) >= len(prefix) && s[:len(prefix)] == prefix
		},
	}

	t, err := template.New("admin").Funcs(funcs).Parse(tmpl)
	if err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.Execute(w, items)
}

