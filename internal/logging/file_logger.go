package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/chybatronik/goUserAPI/internal/config"
)

// FileLogger provides file-based logging functionality
type FileLogger struct {
	infoLogger    *log.Logger
	errorLogger   *log.Logger
	debugLogger   *log.Logger
	logFile       *os.File
	errorLogFile  *os.File
	debugLogFile  *os.File
	config        *config.LoggingConfig
	multiWriter   io.Writer
}

// NewFileLogger creates a new file logger instance
func NewFileLogger(cfg *config.LoggingConfig) (*FileLogger, error) {
	logger := &FileLogger{
		config: cfg,
	}

	// Create log directory if it doesn't exist
	logDir := "/app/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log files
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	// Main log file
	logFileName := filepath.Join(logDir, fmt.Sprintf("app_%s.log", timestamp))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	logger.logFile = logFile

	// Error log file
	errorLogFileName := filepath.Join(logDir, fmt.Sprintf("error_%s.log", timestamp))
	errorLogFile, err := os.OpenFile(errorLogFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}
	logger.errorLogFile = errorLogFile

	// Debug log file (only in debug mode)
	var debugLogFile *os.File
	if cfg.Level == "debug" {
		debugLogFileName := filepath.Join(logDir, fmt.Sprintf("debug_%s.log", timestamp))
		debugFile, err := os.OpenFile(debugLogFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logFile.Close()
			errorLogFile.Close()
			return nil, fmt.Errorf("failed to open debug log file: %w", err)
		}
		debugLogFile = debugFile
		logger.debugLogFile = debugLogFile
	}

	// Create loggers with different destinations
	logger.infoLogger = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.errorLogger = log.New(io.MultiWriter(errorLogFile, os.Stderr), "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	if debugLogFile != nil {
		logger.debugLogger = log.New(debugLogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	// Create multi-writer for both file and stdout output (for Docker logging)
	logger.multiWriter = io.MultiWriter(logFile, os.Stdout)

	return logger, nil
}

// LogInfo logs an info message
func (l *FileLogger) LogInfo(format string, v ...interface{}) {
	if l.infoLogger != nil {
		l.infoLogger.Printf(format, v...)
	}
	// Also log to stdout for Docker
	fmt.Printf("INFO: "+format+"\n", v...)
}

// LogError logs an error message
func (l *FileLogger) LogError(format string, v ...interface{}) {
	if l.errorLogger != nil {
		l.errorLogger.Printf(format, v...)
	}
}

// LogDebug logs a debug message (only if debug level is enabled)
func (l *FileLogger) LogDebug(format string, v ...interface{}) {
	if l.config.Level == "debug" && l.debugLogger != nil {
		l.debugLogger.Printf(format, v...)
	}
	// Also log debug to stdout for Docker when debug enabled
	if l.config.Level == "debug" {
		fmt.Printf("DEBUG: "+format+"\n", v...)
	}
}

// LogWarning logs a warning message
func (l *FileLogger) LogWarning(format string, v ...interface{}) {
	if l.infoLogger != nil {
		l.infoLogger.Printf("WARNING: "+format, v...)
	}
	// Also log to stdout for Docker
	fmt.Printf("WARNING: "+format+"\n", v...)
}

// Close closes all log files
func (l *FileLogger) Close() error {
	var errs []error

	if l.logFile != nil {
		if err := l.logFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close log file: %w", err))
		}
	}

	if l.errorLogFile != nil {
		if err := l.errorLogFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close error log file: %w", err))
		}
	}

	if l.debugLogFile != nil {
		if err := l.debugLogFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close debug log file: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors closing log files: %v", errs)
	}

	return nil
}

// GetLogFiles returns list of current log files
func (l *FileLogger) GetLogFiles() ([]string, error) {
	logDir := "/app/logs"
	files, err := os.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() {
			logFiles = append(logFiles, filepath.Join(logDir, file.Name()))
		}
	}

	return logFiles, nil
}

// CleanupOldLogs removes log files older than specified days
func (l *FileLogger) CleanupOldLogs(days int) error {
	logDir := "/app/logs"
	files, err := os.ReadDir(logDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(logDir, file.Name())
		fileInfo, err := file.Info()
		if err != nil {
			continue // Skip files we can't get info for
		}

		if fileInfo.ModTime().Before(cutoffTime) {
			if err := os.Remove(filePath); err != nil {
				// Log the error but don't fail the operation
				l.LogWarning("Failed to remove old log file %s: %v", filePath, err)
			} else {
				l.LogInfo("Removed old log file: %s", filePath)
			}
		}
	}

	return nil
}