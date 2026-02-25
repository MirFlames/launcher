package com.launcher;

/**
 * Интерфейс для отображения прогресса запуска Minecraft.
 * Позволяет обновлять этап, статус и процент выполнения.
 */
public interface LaunchProgress {
    /**
     * Устанавливает текущий этап (например, "Проверка модов").
     */
    void setStage(String stage);

    /**
     * Устанавливает детальный статус (например, "Скачивание mod.jar 45%").
     */
    void setStatus(String status);

    /**
     * Устанавливает общий прогресс 0.0–1.0.
     */
    void setProgress(double progress);

    /**
     * Режим неопределённого прогресса (анимация загрузки).
     */
    void setIndeterminate(boolean indeterminate);

    /**
     * Завершение с успехом — скрыть overlay.
     */
    void done();

    /**
     * Завершение с ошибкой — показать сообщение и скрыть overlay.
     */
    void fail(String errorMessage);
}
