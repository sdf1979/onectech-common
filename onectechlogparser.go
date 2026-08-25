package onectechcommon

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/icza/backscanner"
)

var BOM = []byte{0xEF, 0xBB, 0xBF}

type FileLog struct {
	name        string
	tm          time.Time
	file        *os.File
	reader      *bufio.Reader
	event       []byte
	bufEOF      []byte
	lastReadEOF time.Time
}

func NewFileLog(name string) *FileLog {
	f := &FileLog{name: name}
	f.setTime()
	return f
}

func (f *FileLog) Name() string {
	return f.name
}

func (f *FileLog) Time() time.Time {
	return f.tm
}

func (f *FileLog) Open() error {
	if f.file != nil {
		return errors.New("file already opened")
	}
	var err error
	f.file, err = os.OpenFile(f.name, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defLog.Debugf("file open %s\n", f.file.Name())

	f.event = make([]byte, 0, 64*1024)
	f.reader = bufio.NewReaderSize(f.file, 64*1024)

	return nil
}

func (f *FileLog) Close() error {
	if f.file == nil {
		return nil
	}
	err := f.file.Close()

	defLog.Debugf("file close %s\n", f.file.Name())

	f.tm = time.Time{}
	f.file = nil
	f.reader = nil
	f.event = nil

	return err
}

func (f *FileLog) Size() (int64, error) {
	size := int64(-1)

	file, err := os.OpenFile(f.name, os.O_RDONLY, 0)
	if err != nil {
		return size, err
	}
	defer file.Close()

	size, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return size, err
	}

	return size, nil
}

func (f *FileLog) SkipBom() {
	peek, err := f.reader.Peek(3)
	if err == nil && bytes.Equal(BOM, peek) {
		_, _ = f.reader.Discard(3)
	}
}

func (f *FileLog) SeekLastEvent() (int64, error) {
	size := int64(-1)

	file, err := os.OpenFile(f.name, os.O_RDONLY, 0)
	if err != nil {
		return size, err
	}
	defer file.Close()

	size, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return size, err
	}

	scanner := backscanner.New(file, int(size))
	for {
		line, pos, err := scanner.LineBytes()
		if err != nil {
			return size, nil
		}

		if isNewEvent(line) {
			return int64(pos), nil
		}
	}
}

func (f *FileLog) SetPos(pos int64) error {
	_, err := f.file.Seek(pos, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileLog) Read(ctx context.Context, out chan<- *EventLog, isTail bool) (int64, error) {
	readBytes := int64(0)
	for {
		if ctx.Err() != nil {
			f.Close()
			return 0, nil
		}
		data, err := f.reader.ReadBytes('\n')
		readBytes += int64(len(data))
		if err != nil {
			if err == io.EOF {
				if isTail {
					f.bufEOF = append(f.bufEOF, data...)
					now := time.Now()
					if f.lastReadEOF.IsZero() {
						f.lastReadEOF = now
					}
					if now.Sub(f.lastReadEOF).Seconds() >= 10 {
						f.sendEvent(out)
						f.lastReadEOF = time.Time{}
					}
				} else {
					f.sendEvent(out)
				}
				break
			}
			return readBytes, err
		} else {
			f.lastReadEOF = time.Time{}
			if len(f.bufEOF) > 0 {
				dataTmp := make([]byte, 0, len(f.bufEOF)+len(data))
				dataTmp = append(dataTmp, f.bufEOF...)
				dataTmp = append(dataTmp, data...)
				f.bufEOF = f.bufEOF[:0]

				data = dataTmp
			}

			if isNewEvent(data) {
				f.sendEvent(out)
			}
			f.event = append(f.event, data...)
		}
	}

	return readBytes, nil
}

func (f *FileLog) setTime() {
	baseName := []rune(filepath.Base(f.name))
	if len(baseName) >= 8 {
		tm, err := time.ParseInLocation("06010215", string(baseName[:8]), time.Local)
		if err != nil {
			return
		}
		f.tm = tm
	}
}

func (f *FileLog) sendEvent(out chan<- *EventLog) {
	if len(f.event) > 0 {
		eventLog := NewEvent(f.tm, f.event)
		out <- eventLog
	}
	f.event = f.event[:0]
}

func isNewEvent(data []byte) bool {

	if len(data) < 14 {
		return false
	}

	//position 3
	if data[2] != ':' {
		return false
	}

	//position 6
	if data[5] != '.' {
		return false
	}

	//position 13
	if data[12] != '-' {
		return false
	}

	//position 0,1
	if data[0] < '0' || data[0] > '9' || data[1] < '0' || data[1] > '9' {
		return false
	}

	//position 3,4
	if data[3] < '0' || data[3] > '9' || data[4] < '0' || data[4] > '9' {
		return false
	}

	//position 6..11
	for i := 6; i < 12; i++ {
		if data[i] < '0' || data[i] > '9' {
			return false
		}
	}

	//position 14
	if data[13] < '0' || data[13] > '9' {
		return false
	}

	return true
}
