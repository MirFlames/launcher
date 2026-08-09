package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tarEntry — описание записи для синтетического tar.gz.
type tarEntry struct {
	name     string
	body     string
	mode     int64
	typeflag byte
	linkname string
}

func buildTarGz(t *testing.T, dir string, entries []tarEntry) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		flag := e.typeflag
		if flag == 0 {
			flag = tar.TypeReg
		}
		mode := e.mode
		if mode == 0 {
			mode = 0644
		}
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     mode,
			Size:     int64(len(e.body)),
			Typeflag: flag,
			Linkname: e.linkname,
		}
		if flag != tar.TypeReg {
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader(%s): %v", e.name, err)
		}
		if flag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("Write(%s): %v", e.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "jdk.tar.gz")
	if err := os.WriteFile(p, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func buildZip(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "jdk.zip")
	if err := os.WriteFile(p, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Windows-раскладка: root/bin/java.exe → <target>/bin/java.exe
func TestExtractJDKArchive_ZipWindowsLayout(t *testing.T) {
	tmp := t.TempDir()
	archive := buildZip(t, tmp, map[string]string{
		// Запись самого корневого каталога реальные архивы Adoptium содержат —
		// она не должна мешать вычислению срезаемого префикса.
		"jdk-21.0.5+11/":             "",
		"jdk-21.0.5+11/bin/java.exe": "binary",
		"jdk-21.0.5+11/lib/rt.jar":   "jar",
	})
	target := filepath.Join(tmp, "out")

	if err := extractJDKArchive(archive, target, nil); err != nil {
		t.Fatalf("extractJDKArchive: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "bin", "java.exe"))
	if err != nil {
		t.Fatalf("корень архива не срезан, bin/java.exe отсутствует: %v", err)
	}
	if string(got) != "binary" {
		t.Errorf("содержимое = %q, ожидалось %q", got, "binary")
	}
	if _, err := os.Stat(filepath.Join(target, "lib", "rt.jar")); err != nil {
		t.Errorf("lib/rt.jar отсутствует: %v", err)
	}
}

// Linux-раскладка: root/bin/java с exec-битом → <target>/bin/java, бит сохранён.
func TestExtractJDKArchive_TarGzLinuxLayout(t *testing.T) {
	tmp := t.TempDir()
	archive := buildTarGz(t, tmp, []tarEntry{
		{name: "jdk-21.0.5+11/", typeflag: tar.TypeDir, mode: 0755},
		{name: "jdk-21.0.5+11/bin/java", body: "binary", mode: 0755},
		{name: "jdk-21.0.5+11/lib/rt.jar", body: "jar", mode: 0644},
	})
	target := filepath.Join(tmp, "out")

	if err := extractJDKArchive(archive, target, nil); err != nil {
		t.Fatalf("extractJDKArchive: %v", err)
	}
	javaPath := filepath.Join(target, "bin", "java")
	st, err := os.Stat(javaPath)
	if err != nil {
		t.Fatalf("bin/java отсутствует: %v", err)
	}
	// Без exec-бита java не запустится вовсе — ради этого и правка.
	if runtime.GOOS != "windows" && st.Mode().Perm()&0111 == 0 {
		t.Errorf("exec-бит потерян: режим = %v", st.Mode().Perm())
	}
}

// macOS-раскладка: бандл root/Contents/Home/... → <target>/bin/java,
// служебные файлы бандла отброшены.
func TestExtractJDKArchive_TarGzMacBundleLayout(t *testing.T) {
	tmp := t.TempDir()
	archive := buildTarGz(t, tmp, []tarEntry{
		{name: "jdk-21.0.5+11/Contents/Info.plist", body: "plist", mode: 0644},
		{name: "jdk-21.0.5+11/Contents/MacOS/libjli.dylib", body: "dylib", mode: 0755},
		{name: "jdk-21.0.5+11/Contents/Home/bin/java", body: "binary", mode: 0755},
		{name: "jdk-21.0.5+11/Contents/Home/lib/rt.jar", body: "jar", mode: 0644},
	})
	target := filepath.Join(tmp, "out")

	if err := extractJDKArchive(archive, target, nil); err != nil {
		t.Fatalf("extractJDKArchive: %v", err)
	}
	// Именно этот путь ждёт лаунчер после нормализации java_executable.
	got, err := os.ReadFile(filepath.Join(target, "bin", "java"))
	if err != nil {
		t.Fatalf("Contents/Home не срезан, bin/java отсутствует: %v", err)
	}
	if string(got) != "binary" {
		t.Errorf("содержимое = %q, ожидалось %q", got, "binary")
	}
	if _, err := os.Stat(filepath.Join(target, "Contents")); !os.IsNotExist(err) {
		t.Errorf("служебные файлы бандла не должны попадать в JAVA_HOME")
	}
}

// Симлинки внутри JDK должны переноситься (в архивах mac/linux они есть).
func TestExtractJDKArchive_TarGzSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("симлинки на Windows требуют привилегий")
	}
	tmp := t.TempDir()
	archive := buildTarGz(t, tmp, []tarEntry{
		{name: "jdk-21/bin/java", body: "binary", mode: 0755},
		{name: "jdk-21/bin/java-alias", typeflag: tar.TypeSymlink, linkname: "java"},
	})
	target := filepath.Join(tmp, "out")

	if err := extractJDKArchive(archive, target, nil); err != nil {
		t.Fatalf("extractJDKArchive: %v", err)
	}
	link, err := os.Readlink(filepath.Join(target, "bin", "java-alias"))
	if err != nil {
		t.Fatalf("симлинк не создан: %v", err)
	}
	if link != "java" {
		t.Errorf("цель симлинка = %q, ожидалось %q", link, "java")
	}
}

// Формат определяется по сигнатуре, а не по расширению: tar.gz под именем .zip
// должен распаковаться — ровно этот случай и ломал macOS.
func TestExtractJDKArchive_SniffsFormatNotExtension(t *testing.T) {
	tmp := t.TempDir()
	archive := buildTarGz(t, tmp, []tarEntry{
		{name: "jdk-21/bin/java", body: "binary", mode: 0755},
	})
	misnamed := filepath.Join(tmp, "jdk.zip")
	if err := os.Rename(archive, misnamed); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(tmp, "out")

	if err := extractJDKArchive(misnamed, target, nil); err != nil {
		t.Fatalf("tar.gz с расширением .zip должен распаковаться, получено: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "bin", "java")); err != nil {
		t.Errorf("bin/java отсутствует: %v", err)
	}
}

// Мусор вместо архива должен давать диагностируемую ошибку,
// а не голое "zip: not a valid zip file".
func TestExtractJDKArchive_UnknownFormatIsDiagnosable(t *testing.T) {
	tmp := t.TempDir()
	junk := filepath.Join(tmp, "jdk.zip")
	if err := os.WriteFile(junk, []byte("<html>404 Not Found</html>"), 0644); err != nil {
		t.Fatal(err)
	}

	err := extractJDKArchive(junk, filepath.Join(tmp, "out"), nil)
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}
	msg := err.Error()
	if !strings.Contains(msg, "26") {
		t.Errorf("ошибка должна называть размер файла, получено: %s", msg)
	}
	if !strings.Contains(msg, "сигнатур") {
		t.Errorf("ошибка должна называть сигнатуру, получено: %s", msg)
	}
}

// Запись, уводящая за пределы папки установки (zip-slip), должна отвергаться.
func TestExtractJDKArchive_RejectsPathTraversal(t *testing.T) {
	tmp := t.TempDir()
	archive := buildTarGz(t, tmp, []tarEntry{
		{name: "jdk-21/bin/java", body: "ok", mode: 0755},
		{name: "jdk-21/../../evil", body: "pwned", mode: 0644},
	})
	target := filepath.Join(tmp, "out")

	// Запись вне корня отсекается ещё на срезе префикса либо проверкой пути;
	// в любом случае файл не должен появиться за пределами target.
	_ = extractJDKArchive(archive, target, nil)
	if _, err := os.Stat(filepath.Join(tmp, "evil")); !os.IsNotExist(err) {
		t.Error("запись вырвалась за пределы папки установки")
	}
}

func TestNormalizeJDKInfo(t *testing.T) {
	info := &JDKInfo{JavaExecutable: `bin\java.exe`}
	normalizeJDKInfo(info)

	if runtime.GOOS == "windows" {
		if info.JavaExecutable != `bin\java.exe` {
			t.Errorf("на Windows значение менять не нужно, получено %q", info.JavaExecutable)
		}
		return
	}
	// Бэкенд отдаёт Windows-путь всем клиентам — .exe должен быть срезан.
	if info.JavaExecutable != `bin\java` {
		t.Errorf("java_executable = %q, ожидалось %q", info.JavaExecutable, `bin\java`)
	}
}

func TestJDKStripPrefix(t *testing.T) {
	cases := []struct {
		name  string
		names []string
		want  string
	}{
		{
			name:  "windows/linux: один корневой каталог",
			names: []string{"jdk-21/bin/java", "jdk-21/lib/rt.jar"},
			want:  "jdk-21/",
		},
		{
			name:  "macos: бандл, берём только JAVA_HOME",
			names: []string{"jdk-21/Contents/Info.plist", "jdk-21/Contents/Home/bin/java"},
			want:  "jdk-21/Contents/Home/",
		},
		{
			name:  "нет общего корня — не срезаем",
			names: []string{"a/bin/java", "b/lib/rt.jar"},
			want:  "",
		},
		{
			name:  "файл в корне — не срезаем",
			names: []string{"java", "jdk-21/bin/java"},
			want:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := jdkStripPrefix(tc.names); got != tc.want {
				t.Errorf("jdkStripPrefix() = %q, ожидалось %q", got, tc.want)
			}
		})
	}
}
