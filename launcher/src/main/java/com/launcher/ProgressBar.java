package com.launcher;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.*;
import javax.swing.plaf.ProgressBarUI;
import javax.swing.plaf.basic.BasicProgressBarUI;
import java.awt.*;
import java.awt.event.WindowAdapter;
import java.awt.event.WindowEvent;

/**
 * Универсальный прогресс-бар для установки JDK, модов и версий игры
 * Блокирует запуск игры во время операций
 */
public class ProgressBar extends JDialog {
    private static final Logger log = LoggerFactory.getLogger(ProgressBar.class);
    
    private JProgressBar progressBar;
    private JLabel statusLabel;
    private boolean cancelled = false;
    
    public ProgressBar(JFrame parent, String title) {
        super(parent, title, false); // Немодальное: скачивание/установка идут в фоне, EDT свободен для отрисовки
        setDefaultCloseOperation(DO_NOTHING_ON_CLOSE); // Нельзя закрыть
        
        addWindowListener(new WindowAdapter() {
            @Override
            public void windowClosing(WindowEvent e) {
                // Игнорируем закрытие - нельзя отменить
            }
        });
        
        initializeUI();
        pack();
        setLocationRelativeTo(parent);
    }
    
    private void initializeUI() {
        setLayout(new BorderLayout(10, 10));
        setResizable(false);
        
        // Статусная метка
        statusLabel = new JLabel("Инициализация...");
        statusLabel.setHorizontalAlignment(SwingConstants.CENTER);
        add(statusLabel, BorderLayout.NORTH);
        
        // Прогресс-бар: подкласс с переопределённым updateUI(), т.к. конструктор
        // JProgressBar вызывает updateUI() → UIManager.getUI() до нашего setUI().
        // В native-image lookup падает. Ставим BasicProgressBarUI напрямую в updateUI().
        progressBar = new JProgressBar(0, 100) {
            @Override
            public void updateUI() {
                setUI((ProgressBarUI) BasicProgressBarUI.createUI(this));
            }
        };
        progressBar.setStringPainted(true);
        progressBar.setString("0%");
        add(progressBar, BorderLayout.CENTER);
        
        // Отступы
        ((JPanel) getContentPane()).setBorder(BorderFactory.createEmptyBorder(20, 20, 20, 20));
    }
    
    /**
     * Обновляет прогресс (0.0 - 1.0)
     */
    public void setProgress(double progress) {
        int value = (int) (progress * 100);
        progressBar.setValue(value);
        progressBar.setString(value + "%");
    }
    
    /**
     * Устанавливает текст статуса
     */
    public void setStatus(String status) {
        statusLabel.setText(status);
        log.info("Progress: {} - {}", status, progressBar.getValue() + "%");
    }
    
    /**
     * Режим «неопределённый» прогресс (полоска в движении), когда размер неизвестен
     */
    public void setIndeterminate(boolean indeterminate) {
        progressBar.setIndeterminate(indeterminate);
    }
    
    /**
     * Показывает прогресс-бар (блокирующий вызов)
     */
    public void showProgress() {
        SwingUtilities.invokeLater(() -> {
            setVisible(true);
        });
    }
    
    /**
     * Скрывает прогресс-бар
     */
    public void hideProgress() {
        SwingUtilities.invokeLater(() -> {
            setVisible(false);
            dispose();
        });
    }
    
    /**
     * Показывает ошибку и закрывает прогресс-бар
     */
    public void showError(String errorMessage) {
        SwingUtilities.invokeLater(() -> {
            JOptionPane.showMessageDialog(this, errorMessage, "Ошибка", JOptionPane.ERROR_MESSAGE);
            setVisible(false);
            dispose();
        });
    }
}
