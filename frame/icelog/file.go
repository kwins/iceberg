package icelog

import (
	"bytes"
	"os"
	"path"
	"time"
)

var pathVariableTable map[byte]func(*time.Time) int

// FileWriter 实现Write接口
type FileWriter struct {
	filePath string
	file     *os.File
	buf      *bytes.Buffer
}

// NewFileWriter new file writer
func NewFileWriter(path string) *FileWriter {
	fw := new(FileWriter)
	fw.buf = bytes.NewBuffer(nil)
	fw.filePath = path
	fw.Rotate()
	fw.initWriter()
	return fw
}

func (w *FileWriter) initWriter() {
	go func() {
		flushTimer := time.NewTimer(time.Millisecond * 500)
		rotateTimer := time.NewTimer(time.Second * 10)
		for {
			select {
			case <-flushTimer.C:
				if w.file != nil {
					w.file.Write(w.buf.Next(w.buf.Len()))
				}
				flushTimer.Reset(time.Millisecond * 500)
			case <-rotateTimer.C:
				w.Rotate()
				rotateTimer.Reset(time.Second * 10)
			}
		}
	}()
}

func (w *FileWriter) Write(text string) error {
	_, err := w.buf.WriteString(text)
	return err
}

// Rotate Rotate
func (w *FileWriter) Rotate() error {
	oldFile := w.file

	if err := os.MkdirAll(path.Dir(w.filePath), 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	path := w.filePath + "." + time.Now().Format("2006-01-02")

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w.file = f

	if oldFile != nil {
		oldFile.Close()
	}
	return nil
}

// Close close
func (w *FileWriter) Close() error {
	_, err := w.file.Write(w.buf.Bytes())
	err = w.file.Close()
	return err
}

// TO DO 日志轮转
func getYear(now *time.Time) int {
	return now.Year()
}

func getMonth(now *time.Time) int {
	return int(now.Month())
}

func getDay(now *time.Time) int {
	return now.Day()
}

func getHour(now *time.Time) int {
	return now.Hour()
}

func getMin(now *time.Time) int {
	return now.Minute()
}

func convertPatternToFmt(pattern []byte) string {
	pattern = bytes.Replace(pattern, []byte("%Y"), []byte("%d"), -1)
	pattern = bytes.Replace(pattern, []byte("%M"), []byte("%02d"), -1)
	pattern = bytes.Replace(pattern, []byte("%D"), []byte("%02d"), -1)
	pattern = bytes.Replace(pattern, []byte("%H"), []byte("%02d"), -1)
	pattern = bytes.Replace(pattern, []byte("%m"), []byte("%02d"), -1)
	return string(pattern)
}

func init() {
	pathVariableTable = make(map[byte]func(*time.Time) int, 5)
	pathVariableTable['Y'] = getYear
	pathVariableTable['M'] = getMonth
	pathVariableTable['D'] = getDay
	pathVariableTable['H'] = getHour
	pathVariableTable['m'] = getMin
}
