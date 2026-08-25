package main

import (
	"github.com/ikermy/air-logger/v2/pkg/logger"
)

func main() {
	// ============ Пример 1: Базовое использование ============
	println("\n=== Пример 1: Базовое использование ===")
	logger.SetPatch("./logs/example1.log").Apply()
	defer logger.Close()

	logger.Debug("Это сообщение DEBUG - НЕ будет залогировано")
	logger.Info("Это сообщение INFO - БУДЕТ залогировано")
	logger.Warn("Это сообщение WARNING - БУДЕТ залогировано")
	logger.Error("Это сообщение ERROR - БУДЕТ залогировано")

	logger.Close()

	// ============ Пример 2: С включенным DEBUG ============
	println("\n=== Пример 2: С включенным DEBUG ===")
	logger.SetPatch("./logs/example2.log").
		WithLogLevel(logger.DEBUG).
		Apply()
	defer logger.Close()

	logger.Debug("Теперь DEBUG - БУДЕТ залогировано")
	logger.Info("INFO - БУДЕТ залогировано")

	logger.Close()

	// ============ Пример 3: Только WARNING и выше ============
	println("\n=== Пример 3: Только WARNING и выше ===")
	logger.SetPatch("./logs/example3.log").
		WithLogLevel(logger.WARNING).
		Apply()
	defer logger.Close()

	logger.Debug("DEBUG - НЕ будет")
	logger.Info("INFO - НЕ будет")
	logger.Warn("WARNING - БУДЕТ")
	logger.Error("ERROR - БУДЕТ")

	logger.Close()

	// ============ Пример 4: Полная конфигурация ============
	println("\n=== Пример 4: Полная конфигурация ===")
	logger.SetPatch("./logs/app.log").
		WithLogLevel(logger.INFO).
		WithMaxSize(10).    // 10 MB
		WithMaxBackups(5).  // Держать 5 резервных копий
		WithMaxAge(90).     // Хранить 90 дней
		WithCompress(true). // Сжимать архивы
		Apply()
	defer logger.Close()

	logger.Info("Приложение запущено")
	logger.Info("Версия: 1.0.0")
	userID := uint32(12345)
	logger.Info("Пользователь вошел", userID)
	logger.Warn("Большой запрос выполняется", userID)

	logger.Close()

	// ============ Пример 5: Сохранение конфига ============
	println("\n=== Пример 5: Сохранение конфига ============")

	// Создаем конфигурацию
	config := logger.SetPatch("./logs/example5.log").
		WithLogLevel(logger.DEBUG).
		WithMaxSize(5)

	// Применяем конфиг
	config.Apply()
	defer logger.Close()

	logger.Debug("Конфиг применен и работает")
	logger.Info("Это INFO сообщение")

	logger.Close()

	// ============ Пример 6: Docker / только stdout ============
	println("\n=== Пример 6: Docker / только stdout ===")
	logger.StdOut().
		WithLogLevel(logger.INFO).
		WithCaller(false).
		Apply()
	defer logger.Close()

	logger.Info("Лог только в stdout для Promtail")
	logger.Warn("Файловая ротация в этом режиме отключена")

	logger.Close()

	println("\n✅ Все примеры выполнены!")
	println("Проверьте папку ./logs/ для файловых режимов и stdout для Docker-режима")
}
