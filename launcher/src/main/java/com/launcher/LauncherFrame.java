package com.launcher;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.*;
import java.awt.*;
import java.awt.event.MouseAdapter;
import java.awt.event.MouseEvent;
import java.awt.image.BufferedImage;
import javax.imageio.ImageIO;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.util.Optional;
import javax.swing.JOptionPane;

public class LauncherFrame extends JFrame {

    private static Logger log;
    private static final Color BACKGROUND_COLOR = new Color(30, 30, 35);
    private static final Color PANEL_COLOR = new Color(40, 40, 45);
    private static final Color ACCENT_COLOR = new Color(0, 120, 215);
    private static final Color TEXT_COLOR = new Color(240, 240, 240);
    private static final Color BUTTON_COLOR = new Color(50, 50, 55);
    private static final Color BUTTON_HOVER = new Color(60, 60, 65);
    private BufferedImage backgroundImage;
    private BufferedImage playButtonBackground;
    private BufferedImage playButton;
    private BufferedImage settingsButton;
    private BufferedImage closeButton;
    private int imageWidth = 900;
    private int imageHeight = 600;
    private Point dragStartPoint;
    private JButton playButtonComponent;

    public LauncherFrame() {
        super("Майнкрафт лаунчер," + " версия " + Consts.LAUNCHER_VERSION);
        log.info("LauncherFrame constructor start");
        loadFavicon();
        log.info("loadFavicon done");
        loadBackgroundImage();
        log.info("loadBackgroundImage done");
        loadButtonImages();
        log.info("loadButtonImages done");
        initializeUI();
        log.info("initializeUI done");
    }

    private void loadFavicon() {
        try {
            InputStream imageStream = getClass().getResourceAsStream("/images/favicon.png");
            if (imageStream != null) {
                setIconImage(ImageIO.read(imageStream));
            } else {
                // Если не найдено в classpath, пытаемся загрузить из файловой системы
                File imageFile = new File("resources/images/favicon.png");
                if (imageFile.exists()) {
                    setIconImage(ImageIO.read(imageFile));
                }
            }
        } catch (IOException e1) {
            log.warn("loadFavicon: {}", e1.getMessage());
        }
    }

    private void loadBackgroundImage() {
        try {
            // Сначала пытаемся загрузить из classpath (для работы в JAR)
            InputStream imageStream = getClass().getResourceAsStream("/images/background.png");
            if (imageStream != null) {
                backgroundImage = ImageIO.read(imageStream);
                imageStream.close();
            } else {
                // Если не найдено в classpath, пытаемся загрузить из файловой системы
                File imageFile = new File("resources/images/background.png");
                if (imageFile.exists()) {
                    backgroundImage = ImageIO.read(imageFile);
                }
            }
            // Устанавливаем размер окна равным размеру изображения
            if (backgroundImage != null) {
                imageWidth = backgroundImage.getWidth();
                imageHeight = backgroundImage.getHeight();
            }
        } catch (IOException e) {
            log.warn("loadBackgroundImage: {}", e.getMessage());
        }
    }

    private void loadButtonImages() {
        try {
            // Загрузка фона кнопки
            InputStream bgStream = getClass().getResourceAsStream("/images/play_button_background.png");
            if (bgStream != null) {
                playButtonBackground = ImageIO.read(bgStream);
                bgStream.close();
            } else {
                File bgFile = new File("resources/images/play_button_background.png");
                if (bgFile.exists()) {
                    playButtonBackground = ImageIO.read(bgFile);
                }
            }
            
            // Загрузка текста кнопки
            InputStream textStream = getClass().getResourceAsStream("/images/play_button.png");
            if (textStream != null) {
                playButton = ImageIO.read(textStream);
                textStream.close();
            } else {
                File textFile = new File("resources/images/play_button.png");
                if (textFile.exists()) {
                    playButton = ImageIO.read(textFile);
                }
            }
            
            // Загрузка кнопки закрытия
            InputStream closeStream = getClass().getResourceAsStream("/images/close_button.png");
            if (closeStream != null) {
                closeButton = ImageIO.read(closeStream);
                closeStream.close();
            } else {
                File closeFile = new File("resources/images/close_button.png");
                if (closeFile.exists()) {
                    closeButton = ImageIO.read(closeFile);
                }
            }

            // Загрузка кнопки настроек
            InputStream settingsStream = getClass().getResourceAsStream("/images/settings_button.png");
            if (settingsStream != null) {
                settingsButton = ImageIO.read(settingsStream);
                settingsStream.close();
            } else {
                File settingsFile = new File("resources/images/settings_button.png");
                if (settingsFile.exists()) {
                    settingsButton = ImageIO.read(settingsFile);
                }
            }
        } catch (IOException e) {
            log.warn("loadButtonImages: {}", e.getMessage());
        }
    }

    private void initializeUI() {
        setDefaultCloseOperation(JFrame.EXIT_ON_CLOSE);
        setUndecorated(true);
        setSize(imageWidth, imageHeight);
        setLocationRelativeTo(null);
        setResizable(false);
        
        // Установка темной темы
        try {
            UIManager.setLookAndFeel(UIManager.getSystemLookAndFeelClassName());
        } catch (Exception e) {
            log.warn("setLookAndFeel: {}", e.getMessage());
        }

        // Главная панель с фоновым изображением
        JPanel mainPanel = getMainPanel();

        // Добавление возможности перетаскивания окна
        mainPanel.addMouseListener(new MouseAdapter() {
            @Override
            public void mousePressed(MouseEvent e) {
                dragStartPoint = e.getPoint();
            }
        });
        
        mainPanel.addMouseMotionListener(new MouseAdapter() {
            @Override
            public void mouseDragged(MouseEvent e) {
                if (dragStartPoint != null) {
                    Point currentLocation = getLocation();
                    setLocation(
                        currentLocation.x + e.getX() - dragStartPoint.x,
                        currentLocation.y + e.getY() - dragStartPoint.y
                    );
                }
            }
        });

        // Создание кнопки играть
        JButton playButton = getPlayButton();

        // Создание кнопки настроек
        JButton settingsButton = getSettingsButton();

        // Обработчик нажатия
        settingsButton.addActionListener(e -> {
            JOptionPane.showMessageDialog(this, "Заглушка: Кнопка настроек нажата!");
        });

        // Создание кнопки закрытия
        JButton closeButton = getCloseButton();

        // Обработчик нажатия - закрытие приложения
        closeButton.addActionListener(e -> {
            System.exit(0);
        });

        mainPanel.add(playButton);
        mainPanel.add(closeButton);
        mainPanel.add(settingsButton);

        add(mainPanel);
    }

    private JPanel getMainPanel() {
        JPanel mainPanel = new JPanel(new BorderLayout()) {
            @Override
            protected void paintComponent(Graphics g) {
                super.paintComponent(g);
                if (backgroundImage != null) {
                    // Рисуем изображение в оригинальном размере без растяжения
                    g.drawImage(backgroundImage, 0, 0, null);
                } else {
                    g.setColor(BACKGROUND_COLOR);
                    g.fillRect(0, 0, getWidth(), getHeight());
                }
                
                // Рисуем версию лаунчера в левом нижнем углу белым цветом
                g.setColor(Color.WHITE);
                g.setFont(g.getFont().deriveFont(Font.PLAIN, 12f));
                String launcherVersion = "v" + Consts.LAUNCHER_VERSION;
                String gameVersion = "v" + "1.0.0";
                FontMetrics fm = g.getFontMetrics();
                int x = 10; // Отступ слева
                int y = getHeight() - 10; // Отступ снизу
                g.drawString(launcherVersion + " - " + gameVersion, x, y);
            }
        };
        mainPanel.setOpaque(false);
        mainPanel.setBorder(null);
        mainPanel.setLayout(null);
        return mainPanel;
    }

    private JButton getSettingsButton() {
        JButton settingsButton = new JButton() {
            @Override
            protected void paintComponent(Graphics g) {
                super.paintComponent(g);
                if (LauncherFrame.this.settingsButton != null) {
                    g.drawImage(LauncherFrame.this.settingsButton, 0, 0, getWidth(), getHeight(), null);
                }
            }
        };

        settingsButton.setContentAreaFilled(false);
        settingsButton.setBorderPainted(false);
        settingsButton.setFocusPainted(false);
        settingsButton.setOpaque(false);

        // Установка размера кнопки
        settingsButton.setBounds(0, 0, 57, 57);

        // Позиция кнопки (как в макете)
        settingsButton.setLocation(imageWidth - settingsButton.getWidth() - 20, imageHeight - settingsButton.getHeight() - 20);
        return settingsButton;
    }

    private JButton getPlayButton() {
        JButton playButton = new JButton() {
            @Override
            protected void paintComponent(Graphics g) {
                super.paintComponent(g);
                if (playButtonBackground != null) {
                    g.drawImage(playButtonBackground, 0, 0, getWidth(), getHeight(), null);
                }
                if (LauncherFrame.this.playButton != null) {
                    int x = (getWidth() - LauncherFrame.this.playButton.getWidth()) / 2 - 50;
                    int y = (getHeight() - LauncherFrame.this.playButton.getHeight()) / 2;
                    g.drawImage(LauncherFrame.this.playButton, x, y, null);
                }
            }
        };

        playButton.setContentAreaFilled(false);
        playButton.setBorderPainted(false);
        playButton.setFocusPainted(false);
        playButton.setOpaque(false);

        // Установка размера кнопки
        if (playButtonBackground != null) {
            playButton.setBounds(0, 0, playButtonBackground.getWidth(), playButtonBackground.getHeight());
        } else {
            playButton.setBounds(0, 0, 200, 60);
        }

        // Позиция кнопки (как в макете)
        playButton.setLocation(22, 318);

        // Обработчик нажатия: запуск в фоне, чтобы EDT не блокировался и прогресс JDK отображался
        playButton.addActionListener(e -> {
            JButton btn = (JButton) e.getSource();
            btn.setEnabled(false);
            Thread worker = new Thread(() -> {
                try {
                    MinecraftLauncher.startMinecraft(LauncherFrame.this);
                } catch (IOException ex) {
                    log.error("Ошибка запуска Minecraft: {}", ex.getMessage(), ex);
                    SwingUtilities.invokeLater(() -> {
                        JOptionPane.showMessageDialog(LauncherFrame.this,
                            "Ошибка запуска Minecraft: " + ex.getMessage(),
                            "Ошибка", JOptionPane.ERROR_MESSAGE);
                    });
                } catch (Exception ex) {
                    log.error("Ошибка запуска Minecraft: {}", ex.getMessage(), ex);
                    SwingUtilities.invokeLater(() -> {
                        JOptionPane.showMessageDialog(LauncherFrame.this,
                            "Ошибка запуска Minecraft: " + ex.getMessage(),
                            "Ошибка", JOptionPane.ERROR_MESSAGE);
                    });
                } finally {
                    SwingUtilities.invokeLater(() -> btn.setEnabled(true));
                }
            }, "minecraft-start");
            worker.start();
        });
        playButtonComponent = playButton;
        return playButton;
    }

    private JButton getCloseButton() {
        JButton closeButton = new JButton() {
            @Override
            protected void paintComponent(Graphics g) {
                super.paintComponent(g);
                if (LauncherFrame.this.closeButton != null) {
                    g.drawImage(LauncherFrame.this.closeButton, 0, 0, getWidth(), getHeight(), null);
                }
            }
        };

        closeButton.setContentAreaFilled(false);
        closeButton.setBorderPainted(false);
        closeButton.setFocusPainted(false);
        closeButton.setOpaque(false);

        // Установка размера и позиции кнопки закрытия
        if (this.closeButton != null) {
            closeButton.setBounds(0, 0, this.closeButton.getWidth(), this.closeButton.getHeight());
        } else {
            closeButton.setBounds(0, 0, 30, 30);
        }

        // Размещение в верхнем правом углу
        int closeX = imageWidth - closeButton.getWidth() - 7;
        int closeY = 7;
        closeButton.setLocation(closeX, closeY);

        return closeButton;
    }

    /**
     * Путь к launcher.log: каталог exe (при запуске двойным щелчком user.dir часто System32),
     * иначе user.dir, иначе user.home.
     */
    /**
     * Проверяет и показывает информационное уведомление о завершении обновления
     */
    private static void checkAndShowUpdateNotification(LauncherFrame frame) {
        try {
            File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
            File backupFile = new File(minecraftFolder, "launcher.exe.backup");
            
            // Если есть бэкап - значит было обновление
            if (backupFile.exists()) {
                SwingUtilities.invokeLater(() -> {
                    JOptionPane.showMessageDialog(frame,
                        "Лаунчер успешно обновлен до версии " + Consts.LAUNCHER_VERSION + ".",
                        "Обновление завершено",
                        JOptionPane.INFORMATION_MESSAGE);
                });
                log.info("Показано уведомление о завершении обновления");
                log.info("Удаляем бэкап launcher.exe.backup");
                backupFile.delete();
            }
        } catch (Exception e) {
            log.warn("Не удалось проверить обновление: {}", e.getMessage());
        }
    }
    
    private static File resolveLogFile() {
        try {
            Optional<String> cmd = ProcessHandle.current().info().command();
            if (cmd.isPresent()) {
                File exe = new File(cmd.get());
                File dir = exe.getParentFile();
                if (dir != null && dir.isDirectory()) {
                    return new File(dir, "launcher.log");
                }
            }
        } catch (Throwable ignored) {
            // ProcessHandle недоступен (редкие JVM / окружения)
        }
        File ud = new File(System.getProperty("user.dir", "."));
        if (ud.isDirectory()) {
            return new File(ud, "launcher.log");
        }
        return new File(System.getProperty("user.home", "."), "launcher.log");
    }

    public static void main(String[] args) {
        File logPath = resolveLogFile();
        System.setProperty("LOG_FILE", logPath.getAbsolutePath());
        
        // Устанавливаем java.home для AWT font configuration (критично для GraalVM native-image на Windows)
        String javaHome = System.getProperty("java.home");
        if (javaHome == null || javaHome.isEmpty()) {
            String javaHomeEnv = System.getenv("JAVA_HOME");
            if (javaHomeEnv != null && !javaHomeEnv.isEmpty()) {
                System.setProperty("java.home", javaHomeEnv);
            }
        }
        
        log = LoggerFactory.getLogger(LauncherFrame.class);

        Thread.setDefaultUncaughtExceptionHandler((t, e) ->
                log.error("Uncaught in {}", t.getName(), e));

        log.info("Starting launcher, java.version={}", System.getProperty("java.version"));
        log.info("user.dir={}", System.getProperty("user.dir"));
        log.info("java.home={}", System.getProperty("java.home"));
        log.info("Log file: {}", logPath);
        
        // Блокировка запуска нескольких экземпляров
        if (UpdateManager.isAnotherInstanceRunning()) {
            log.warn("Другой экземпляр лаунчера уже запущен");
            JOptionPane.showMessageDialog(null, 
                "Лаунчер уже запущен. Закройте предыдущий экземпляр.", 
                "Внимание", JOptionPane.WARNING_MESSAGE);
            System.exit(1);
            return;
        }
        
        // Проверка обновлений ПЕРЕД инициализацией UI
        try {
            log.info("Проверка обновлений лаунчера...");
            UpdateManager.LauncherVersionInfo updateInfo = UpdateManager.checkForUpdates();
            
            if (updateInfo != null) {
                log.info("Найдено обновление: {} -> {}", Consts.LAUNCHER_VERSION, updateInfo.version);
                // Принудительное обновление - запускаем updater и закрываемся
                if (UpdateManager.startUpdate(updateInfo)) {
                    log.info("Updater запущен, закрываем лаунчер");
                    System.exit(0);
                    return;
                } else {
                    // Если updater не запустился - показываем ошибку и закрываемся
                    JOptionPane.showMessageDialog(null, 
                        "Не удалось запустить обновление. Лаунчер будет закрыт.", 
                        "Ошибка", JOptionPane.ERROR_MESSAGE);
                    System.exit(1);
                    return;
                }
            }
        } catch (Exception e) {
            log.error("Ошибка при проверке обновлений: {}", e.getMessage(), e);
            // Если нет интернета или сервер недоступен - показать ошибку и закрыть
            JOptionPane.showMessageDialog(null, 
                "Не удалось подключиться к серверу обновлений.\nПроверьте подключение к интернету.", 
                "Ошибка", JOptionPane.ERROR_MESSAGE);
            System.exit(1);
            return;
        }

        SwingUtilities.invokeLater(() -> {
            try {
                log.info("Swing EDT: creating LauncherFrame");
                LauncherFrame launcherFrame = new LauncherFrame();
                launcherFrame.setVisible(true);
                log.info("LauncherFrame visible");
                
                // Проверить был ли запуск после обновления (проверка наличия флага обновления)
                // Можно использовать временный файл или параметр командной строки
                // Для простоты показываем уведомление если был запуск после обновления
                // (это можно определить по наличию launcher.exe.backup или другому маркеру)
                checkAndShowUpdateNotification(launcherFrame);
                
            } catch (Throwable e) {
                log.error("Swing EDT failed", e);
                throw new RuntimeException(e);
            }
        });
    }
}
