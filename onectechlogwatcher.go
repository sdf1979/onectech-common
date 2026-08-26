package onectechcommon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

type FileWatcher struct {
	dir     string
	out     chan<- *EventLog
	threads int
	isTail  bool
	files   map[string]*FileLog
}

func NewFileWatcher(dir string, out chan<- *EventLog, threads int, isTail bool) *FileWatcher {
	fw := &FileWatcher{}
	fw.out = out
	fw.dir = dir
	fw.threads = threads
	fw.isTail = isTail
	fw.files = make(map[string]*FileLog)

	return fw
}

func (fw *FileWatcher) CheckDir() error {
	info, err := os.Stat(fw.dir)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("directory does not exist:'%s'", fw.dir)
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: '%s'", fw.dir)
	}
	return nil
}

func (fw *FileWatcher) Run(ctx context.Context) (int64, error) {
	var totalBytes int64
	var err error

	if fw.isTail {
		err = fw.tail(ctx)
	} else {
		totalBytes, err = fw.run(ctx)
	}

	if err != nil {
		return 0, err
	}

	return totalBytes, nil
}

func (fw *FileWatcher) Close() {
	for _, fl := range fw.files {
		fl.Close()
	}
}

func (fw *FileWatcher) run(ctx context.Context) (int64, error) {
	err := fw.initFiles()
	if err != nil {
		return 0, err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(fw.threads)

	var totalBytes int64

	for _, fl := range fw.files {
		g.Go(func() error {
			n, err := fl.Read(ctx, fw.out, fw.isTail)
			fl.Close()
			if n > 0 {
				atomic.AddInt64(&totalBytes, n)
			}
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return 0, err
	}

	return totalBytes, nil
}

func (fw *FileWatcher) tail(ctx context.Context) error {
	err := fw.initTailFiles()
	if err != nil {
		return err
	}
	lastUpdateFiles := time.Now()

	for {
		if ctx.Err() != nil {
			return nil
		}

		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(fw.threads)

		var totalBytes int64

		for _, fl := range fw.files {
			g.Go(func() error {
				n, err := fl.Read(ctx, fw.out, fw.isTail)
				if n > 0 {
					atomic.AddInt64(&totalBytes, n)
				}
				return err
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		if time.Since(lastUpdateFiles) >= 30*time.Second {
			fw.addRemoveFiles()
			lastUpdateFiles = time.Now()
			if totalBytes == 0 {
				totalBytes = -1
			}
		}

		if totalBytes == 0 {
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (fw *FileWatcher) addRemoveFiles() error {

	now := time.Now()
	nowHour := time.Now().Truncate(time.Hour)
	isRemove := now.Sub(nowHour).Seconds() >= 10
	for key, fl := range fw.files {
		if nowHour.After(fl.Time()) && isRemove {
			fl.Close()
			delete(fw.files, key)
		}
	}

	files, err := fw.getFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		fl, ok := fw.files[file]
		if !ok {
			fl = NewFileLog(file)
			size, err := fl.Size()
			if err != nil {
				defLog.Debugf("failed to get size of file: %v", err)
				continue
			}
			if size > 3 && nowHour.Equal(fl.Time()) {
				err := fl.Open()
				if err != nil {
					fl.Close()
					defLog.Debugf("failed to find the last event in file %s: %v", fl.Name(), err)
					continue
				}
				fl.SkipBom()
				fw.files[file] = fl
			}
		}
	}

	return nil
}

func (fw *FileWatcher) getFiles() ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(fw.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".log") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (fw *FileWatcher) initFiles() error {
	files, err := fw.getFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		fl, ok := fw.files[file]
		if !ok {
			fl = NewFileLog(file)
			size, err := fl.Size()
			if err != nil {
				defLog.Debugf("failed to get size of file: %v", err)
				continue
			}
			if size > 3 {
				err := fl.Open()
				if err != nil {
					defLog.Debugf("failed to open file: %v", err)
					fl.Close()
					continue
				}
				fl.SkipBom()
				fw.files[file] = fl
			}
		}
	}
	return nil
}

func (fw *FileWatcher) initTailFiles() error {
	files, err := fw.getFiles()
	if err != nil {
		return err
	}

	nowHour := time.Now().Truncate(time.Hour)
	for _, file := range files {
		fl, ok := fw.files[file]
		if !ok {
			fl = NewFileLog(file)
			size, err := fl.Size()
			if err != nil {
				defLog.Debugf("failed to get size of file: %v", err)
				continue
			}
			if size > 3 && nowHour.Equal(fl.Time()) {
				fl.Open()
				pos, err := fl.SeekLastEvent()
				if err != nil {
					defLog.Debugf("failed to find the last event in file %s: %v", fl.Name(), err)
					fl.Close()
					continue
				}
				err = fl.SetPos(pos)
				if err != nil {
					defLog.Debugf("failed to seek in file %s: %v", fl.Name(), err)
					fl.Close()
					continue
				}
				fw.files[file] = fl
			}
		}
	}

	return nil
}
