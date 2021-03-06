package util

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const DATEFORMAT = "2006-01-02"

type UNIT int64

const (
	_       = iota
	KB UNIT = 1 << (iota * 10)
	MB
	GB
	TB
)

// log level
const (
	LOG_DEBUG = iota
	LOG_TRACE
	LOG_INFO
	LOG_WARN
	LOG_ERROR
)

// log level strings
var levels = []string{
	"[DEBUG]",
	"[TRACE]",
	"[INFO]",
	"[WARNING]",
	"[ERROR]",
}

type LoggerInfo struct {
	logLevel      int32
	maxFileSize   int64
	maxFileCount  int32
	dailyRolling  bool
	ifConsoleShow bool
	RollingFile   bool
	logObj        *LogFileInfo
}

type LogFileInfo struct {
	dir        string
	filename   string
	fileSuffix int
	isCover    bool
	fileDate   *time.Time
	mutex      *sync.RWMutex
	logfile    *os.File
	lg         *log.Logger
}

/////////////////////////////////////////////////////////////////////////////////////
type LogInterface interface {
	Debug(format string, a ...interface{})
	Trace(format string, a ...interface{})
	Info(format string, a ...interface{})
	Warn(format string, a ...interface{})
	Error(format string, a ...interface{})
}

type XLog struct {
}

func NewXLogger(logDir string, fileName string, logLevel int, ifConsole int) *XLog {
	createLogDir(logDir)
	setRollingFile(logDir, fileName, 10, 50, MB)
	setLogLevel(int32(logLevel))

	if ifConsole == 1 {
		setConsoleShow(true)
	}

	return &XLog{false, nil}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
var LogInfo LoggerInfo

func init() {
	LogInfo.logLevel = LOG_INFO
	LogInfo.dailyRolling = true
	LogInfo.ifConsoleShow = false
	LogInfo.RollingFile = false
}

func setLogLevel(level int32) {
	LogInfo.logLevel = level
}

func setConsoleShow(ifConsole bool) {
	LogInfo.ifConsoleShow = ifConsole
}

// uint for KB MB GB TB
// file size = maxSize * uint
func setRollingFile(fileDir, fileName string, maxCount int32, maxSize int64, unit UNIT) {
	LogInfo.maxFileCount = maxCount
	LogInfo.maxFileSize = maxSize * int64(unit)
	LogInfo.RollingFile = true
	LogInfo.dailyRolling = false
	LogInfo.logObj = &LogFileInfo{dir: fileDir, filename: fileName, isCover: false, mutex: new(sync.RWMutex)}
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	for i := 1; i <= int(maxCount); i++ {
		if ifExistFile(fileDir + "/" + fileName + "." + strconv.Itoa(i)) {
			LogInfo.logObj.fileSuffix = i
		} else {
			break
		}
	}
	if !LogInfo.logObj.ifMustRename() {
		LogInfo.logObj.logfile, _ = os.OpenFile(fileDir+"/"+fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		LogInfo.logObj.lg = log.New(LogInfo.logObj.logfile, "", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		LogInfo.logObj.rename()
	}
	go fileMonitor()
}

func SetRollingDaily(fileDir, fileName string) {
	LogInfo.RollingFile = false
	LogInfo.dailyRolling = true
	t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
	LogInfo.logObj = &LogFileInfo{dir: fileDir, filename: fileName, fileDate: &t, isCover: false, mutex: new(sync.RWMutex)}
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	if !LogInfo.logObj.ifMustRename() {
		LogInfo.logObj.logfile, _ = os.OpenFile(fileDir+"/"+fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		LogInfo.logObj.lg = log.New(LogInfo.logObj.logfile, "", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		LogInfo.logObj.rename()
	}
}

func consoleF(format string, a ...interface{}) {
	if LogInfo.ifConsoleShow {
		_, file, line, _ := runtime.Caller(2)
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		formatTmp := file + ":" + strconv.Itoa(line) + " " + format
		log.Println(fmt.Sprintf(formatTmp, a...))
	}
}

func catchError() {
	if err := recover(); err != nil {
		log.Println("err", err)
	}
}

func Logf(logLevel int32, format string, a ...interface{}) {
	if LogInfo.logObj.lg == nil {
		log.Println("LogInfo.logObj.lg == nil")
		return
	}
	if LogInfo.dailyRolling {
		fileCheck()
	}
	defer catchError()
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	if LogInfo.logLevel <= logLevel {
		format = levels[logLevel] + " " + format
		LogInfo.logObj.lg.Output(2, fmt.Sprintf(format, a...))
		consoleF(format, a...)
	}
}

func (logFile *LogFileInfo) ifMustRename() bool {
	if LogInfo.dailyRolling {
		t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
		if t.After(*logFile.fileDate) {
			return true
		}
	} else {
		if LogInfo.maxFileCount > 1 {
			if getFileSize(logFile.dir+"/"+logFile.filename) >= LogInfo.maxFileSize {
				return true
			}
		}
	}

	return false
}

func (logFile *LogFileInfo) rename() {
	if LogInfo.dailyRolling {
		fn := logFile.dir + "/" + logFile.filename + "." + logFile.fileDate.Format(DATEFORMAT)
		if !ifExistFile(fn) && logFile.ifMustRename() {
			if logFile.logfile != nil {
				logFile.logfile.Close()
			}
			err := os.Rename(logFile.dir+"/"+logFile.filename, fn)
			if err != nil {
				logFile.lg.Println("rename error, ", err.Error())
			}
			t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
			logFile.fileDate = &t
			logFile.logfile, _ = os.Create(logFile.dir + "/" + logFile.filename)
			logFile.lg = log.New(LogInfo.logObj.logfile, "", log.Ldate|log.Ltime|log.Lshortfile)
		}
	} else {
		logFile.coverNextOne()
	}
}

func (logFile *LogFileInfo) coverNextOne() {
	logFile.fileSuffix = logFile.nextSuffix()
	if logFile.logfile != nil {
		logFile.logfile.Close()
	}
	newFileName := logFile.dir + "/" + logFile.filename + "." + strconv.Itoa(int(logFile.fileSuffix))
	if ifExistFile(newFileName) {
		os.Remove(newFileName)
	}
	os.Rename(logFile.dir+"/"+logFile.filename, newFileName)
	logFile.logfile, _ = os.Create(logFile.dir + "/" + logFile.filename)
	logFile.lg = log.New(LogInfo.logObj.logfile, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func (logFile *LogFileInfo) nextSuffix() int {
	return int(logFile.fileSuffix%int(LogInfo.maxFileCount) + 1)
}

func getFileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		fmt.Println(e.Error())
		return 0
	}
	return f.Size()
}

func ifExistFile(file string) bool {
	_, err := os.Stat(file)
	return err != nil || os.IsExist(err)
}

func fileMonitor() {
	timer := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-timer.C:
			fileCheck()
		}
	}
}

func fileCheck() {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	if LogInfo.logObj != nil && LogInfo.logObj.ifMustRename() {
		LogInfo.logObj.mutex.Lock()
		defer LogInfo.logObj.mutex.Unlock()
		LogInfo.logObj.rename()
	}
}

////////////////////////////////////////////////////////////////
func (xlog *XLog) Debug(format string, a ...interface{}) {
	if LogInfo.logObj.lg == nil {
		log.Println("LogInfo.logObj.lg == nil")
		return
	}
	if LogInfo.dailyRolling {
		fileCheck()
	}
	defer catchError()
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	if LogInfo.logLevel <= LOG_DEBUG {
		format = levels[LOG_DEBUG] + " " + format
		LogInfo.logObj.lg.Output(2, fmt.Sprintf(format, a...))
		consoleF(format, a...)
	}
}

func (xlog *XLog) Trace(format string, a ...interface{}) {
	if LogInfo.logObj.lg == nil {
		log.Println("LogInfo.logObj.lg == nil")
		return
	}
	if LogInfo.dailyRolling {
		fileCheck()
	}
	defer catchError()
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	if LogInfo.logLevel <= LOG_TRACE {
		format = levels[LOG_TRACE] + " " + format
		LogInfo.logObj.lg.Output(2, fmt.Sprintf(format, a...))
		consoleF(format, a...)
	}
}

func (xlog *XLog) Info(format string, a ...interface{}) {
	if LogInfo.logObj.lg == nil {
		log.Println("LogInfo.logObj.lg == nil")
		return
	}
	if LogInfo.dailyRolling {
		fileCheck()
	}
	defer catchError()
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()

	if LogInfo.logLevel <= LOG_INFO {
		format = levels[LOG_INFO] + " " + format
		LogInfo.logObj.lg.Output(2, fmt.Sprintf(format, a...))
		consoleF(format, a...)
	}
}

func (xlog *XLog) Warn(format string, a ...interface{}) {
	if LogInfo.logObj.lg == nil {
		log.Println("LogInfo.logObj.lg == nil")
		return
	}
	if LogInfo.dailyRolling {
		fileCheck()
	}
	defer catchError()
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	if LogInfo.logLevel <= LOG_WARN {
		format = levels[LOG_WARN] + " " + format
		LogInfo.logObj.lg.Output(2, fmt.Sprintf(format, a...))
		consoleF(format, a...)
	}
}

func (xlog *XLog) Error(format string, a ...interface{}) {
	if LogInfo.logObj.lg == nil {
		log.Println("LogInfo.logObj.lg == nil")
		return
	}
	if LogInfo.dailyRolling {
		fileCheck()
	}
	defer catchError()
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	if LogInfo.logLevel <= LOG_ERROR {
		format = levels[LOG_ERROR] + " " + format
		LogInfo.logObj.lg.Output(2, fmt.Sprintf(format, a...))
		consoleF(format, a...)
	}
}

func createLogDir(logDir string) {
	fi, err := os.Stat(logDir)
	if err != nil {
		if os.IsExist(err) == false {
			os.MkdirAll(logDir, 0777)
		}
	} else {
		if fi.IsDir() == false {
			os.MkdirAll(logDir, 0777)
		}
	}
}
