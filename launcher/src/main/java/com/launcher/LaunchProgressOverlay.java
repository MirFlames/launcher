package com.launcher;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.*;
import java.awt.*;

/**
 * Современный overlay прогресса запуска Minecraft.
 * Показывается сразу при нажатии «Играть» и отображает этапы загрузки.
 * Использует fillRect вместо fillRoundRect для совместимости с GraalVM native-image.
 */
public class LaunchProgressOverlay extends JDialog implements LaunchProgress {

    private static final Logger log = LoggerFactory.getLogger(LaunchProgressOverlay.class);

    private static final Color BG_DARK = new Color(30, 30, 38);
    private static final Color PANEL_BG = new Color(45, 45, 55);
    private static final Color ACCENT = new Color(0, 150, 255);
    private static final Color TEXT_MAIN = new Color(240, 240, 245);
    private static final Color TEXT_SECONDARY = new Color(180, 180, 190);

    private JLabel stageLabel;
    private JLabel statusLabel;
    private JPanel progressFill;
    private Timer indeterminateTimer;
    private double currentProgress = 0;
    private boolean indeterminate = true;

    public LaunchProgressOverlay(JFrame parent) {
        super(parent, "", false);
        setUndecorated(true);
        setModal(false);
        setDefaultCloseOperation(DO_NOTHING_ON_CLOSE);
        setBackground(BG_DARK);
        getContentPane().setBackground(BG_DARK);

        initializeUI();
        pack();
        setLocationRelativeTo(parent);
    }

    private void initializeUI() {
        JPanel root = new JPanel(new BorderLayout(0, 0));
        root.setBackground(BG_DARK);
        root.setBorder(BorderFactory.createEmptyBorder(32, 40, 32, 40));

        // Центральная панель
        JPanel center = new JPanel(new BorderLayout(0, 20));
        center.setBackground(BG_DARK);

        stageLabel = new JLabel("Подготовка...");
        stageLabel.setFont(stageLabel.getFont().deriveFont(Font.BOLD, 18f));
        stageLabel.setForeground(TEXT_MAIN);
        stageLabel.setHorizontalAlignment(SwingConstants.CENTER);
        center.add(stageLabel, BorderLayout.NORTH);

        statusLabel = new JLabel("");
        statusLabel.setFont(statusLabel.getFont().deriveFont(Font.PLAIN, 13f));
        statusLabel.setForeground(TEXT_SECONDARY);
        statusLabel.setHorizontalAlignment(SwingConstants.CENTER);
        center.add(statusLabel, BorderLayout.CENTER);

        // Прогресс-бар: fillRect вместо fillRoundRect (native-image не включает sun.java2d.loops.*)
        progressFill = new JPanel() {
            @Override
            protected void paintComponent(Graphics g) {
                int h = getHeight();
                int w = getWidth();
                g.setColor(PANEL_BG);
                g.fillRect(0, 0, w, h);
                if (indeterminate) {
                    int barW = Math.max(60, w / 3);
                    int x = (int) ((System.currentTimeMillis() % 2000) / 2000.0 * (w + barW)) - barW;
                    g.setColor(ACCENT);
                    g.fillRect(Math.max(0, x), 0, barW, h);
                } else {
                    int fillW = (int) (w * Math.min(1.0, Math.max(0.0, currentProgress)));
                    if (fillW > 0) {
                        g.setColor(ACCENT);
                        g.fillRect(0, 0, fillW, h);
                    }
                }
            }
        };
        progressFill.setPreferredSize(new Dimension(320, 8));
        progressFill.setMinimumSize(new Dimension(320, 8));
        progressFill.setBackground(BG_DARK);

        JPanel progressWrapper = new JPanel(new BorderLayout());
        progressWrapper.setBackground(BG_DARK);
        progressWrapper.setBorder(BorderFactory.createEmptyBorder(16, 0, 0, 0));
        progressWrapper.add(progressFill, BorderLayout.CENTER);
        center.add(progressWrapper, BorderLayout.SOUTH);

        root.add(center, BorderLayout.CENTER);
        setContentPane(root);
    }

    @Override
    public void setStage(String stage) {
        SwingUtilities.invokeLater(() -> {
            stageLabel.setText(stage != null ? stage : "");
            log.info("LaunchProgress: stage={}", stage);
        });
    }

    @Override
    public void setStatus(String status) {
        SwingUtilities.invokeLater(() -> {
            statusLabel.setText(status != null ? status : "");
        });
    }

    @Override
    public void setProgress(double progress) {
        SwingUtilities.invokeLater(() -> {
            this.indeterminate = false;
            this.currentProgress = Math.max(0, Math.min(1, progress));
            progressFill.repaint();
        });
    }

    @Override
    public void setIndeterminate(boolean indeterminate) {
        SwingUtilities.invokeLater(() -> {
            this.indeterminate = indeterminate;
            if (indeterminate && indeterminateTimer == null) {
                indeterminateTimer = new Timer(50, e -> progressFill.repaint());
                indeterminateTimer.start();
            } else if (!indeterminate && indeterminateTimer != null) {
                indeterminateTimer.stop();
                indeterminateTimer = null;
            }
            progressFill.repaint();
        });
    }

    @Override
    public void done() {
        SwingUtilities.invokeLater(() -> {
            if (indeterminateTimer != null) {
                indeterminateTimer.stop();
                indeterminateTimer = null;
            }
            setVisible(false);
            dispose();
        });
    }

    @Override
    public void fail(String errorMessage) {
        SwingUtilities.invokeLater(() -> {
            if (indeterminateTimer != null) {
                indeterminateTimer.stop();
                indeterminateTimer = null;
            }
            setVisible(false);
            dispose();
            Window owner = getOwner();
            if (owner != null) {
                JOptionPane.showMessageDialog(owner, errorMessage, "Ошибка запуска", JOptionPane.ERROR_MESSAGE);
            }
        });
    }

    @Override
    public void setVisible(boolean b) {
        if (b) {
            setIndeterminate(true);
            currentProgress = 0;
        }
        super.setVisible(b);
    }
}
