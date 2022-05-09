package util

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

type MyFormatter struct{}

func (m *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	var newLog string

	// HasCaller()为true才会有调用信息
	if entry.HasCaller() {
		fName := filepath.Base(entry.Caller.File)
		if entry.Level == logrus.InfoLevel {
			newLog = fmt.Sprintf("[%s] [%s] %s [%s:%d]\n",
				timestamp, entry.Level, entry.Message, fName, entry.Caller.Line)
		} else {
			newLog = fmt.Sprintf("[%s] [%s] %s\n\t\t [%s:%d %s]\n",
				timestamp, entry.Level, entry.Message, fName, entry.Caller.Line, entry.Caller.Function)
		}
	} else {
		newLog = fmt.Sprintf("[%s] [%s] %s\n", timestamp, entry.Level, entry.Message)
	}

	b.WriteString(newLog)
	return b.Bytes(), nil
}

func InitLog(appName string) {
	// 输出文件名，行号和函数名
	logrus.SetReportCaller(true)

	//设置输出样式，自带的只有两种样式 logrus.JSONFormatter{} 和 logrus.TextFormatter{}，这里自定义一个日志格式化类
	logrus.SetFormatter(&MyFormatter{})
	logrus.SetOutput(os.Stdout)

	//设置 output,默认为 stderr,可以为任何 io.Writer，比如文件 *os.File
	isExists := FileOrPathIsExists("./logs")
	if !isExists {
		os.Mkdir("./logs", os.ModePerm)
	}
	logFile, err := os.OpenFile("./logs/"+appName+"_"+GetToday()+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	writers := []io.Writer{
		logFile,
		os.Stdout}
	fileAndStdoutWriter := io.MultiWriter(writers...)
	if err == nil {
		logrus.SetOutput(fileAndStdoutWriter)
	} else {
		logrus.Info("failed to logs to file.")
	}
	// 设置最低 loglevel，默认 info
	//logrus.SetLevel(logrus.InfoLevel)
}
