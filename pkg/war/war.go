package war

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type (
	WatchAndRun struct {
		watcher     *fsnotify.Watcher
		closeCh     chan struct{}
		runCh       chan struct{}
		cancelRunCh chan cancel
		options     options
		closeMu     sync.Mutex
		// watched 用于保存我们监听了哪些目录, 以及遇到过哪些文件
		watched         map[string]*watchedInfo
		closeWg         sync.WaitGroup
		firstRunSuccess bool
		rootWatched     bool
	}
)

func NewWatchAndRun(opts ...Option) (*WatchAndRun, error) {
	options := options{delay: time.Second, termTimeout: 3 * time.Second, cancelLast: true}
	for _, o := range opts {
		o(&options)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &WatchAndRun{
		watcher:     watcher,
		closeCh:     make(chan struct{}),
		runCh:       make(chan struct{}, 1),
		cancelRunCh: make(chan cancel, 1),
		watched:     make(map[string]*watchedInfo),
		options:     options,
	}, nil
}

func (w *WatchAndRun) Start(context.Context) error {
	if stat, err := os.Stat(w.options.root); err != nil {
		return err
	} else if !stat.IsDir() {
		return errors.New("root is not a directory")
	}
	w.addDir(w.options.root, true, false)
	w.rootWatched = true
	select {
	case w.runCh <- struct{}{}:
	default:
	}
	w.closeWg.Add(2)
	go w.runLoop()
	go w.handleLoop()
	return nil
}

func (w *WatchAndRun) Stop(context.Context) error {
	w.closeMu.Lock()
	defer w.closeMu.Unlock()
	select {
	case <-w.closeCh:
		return nil
	default:
	}
	if err := w.watcher.Close(); err != nil {
		w.logError("close watcher error: %+v", err)
	}
	w.cancelRun()
	close(w.closeCh)
	w.closeWg.Wait()
	return nil
}

func (w *WatchAndRun) addDir(dir string, dfs bool, notifyRun bool) {
	if _, ok := w.watched[dir]; ok {
		w.logError("[BUG] duplicated add dir: %s", dir)
	}
	if err := w.watcher.Add(dir); err != nil {
		w.logError("watch dir error %s %+v", dir, err)
		return
	}
	w.logChange("watch dir: %s", dir)
	w.watched[dir] = &watchedInfo{file: false}
	if dfs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if dir != path {
				if d.IsDir() {
					if !w.shouldWatchDir(path) {
						return filepath.SkipDir
					}
					w.addDir(path, false, notifyRun)
				} else {
					w.maybeAddFile(path, d.Type(), notifyRun)
				}
			}
			return nil
		})
	}
}

func (w *WatchAndRun) maybeAddFile(path string, mode fs.FileMode, notifyRun bool) {
	if (mode&fs.ModeSymlink) == 0 && w.shouldWatchFile(path) {
		if _, ok := w.watched[path]; ok {
			w.logChange("write file %s", path)
		} else {
			w.logChange("watch file %s", path)
			w.watched[path] = &watchedInfo{file: true}
		}
		if notifyRun {
			w.notifyRun()
		}
	}
}

func (w *WatchAndRun) shouldWatchDir(path string) bool {
	// ignore all hidden dirs
	rel, err := filepath.Rel(w.options.root, path)
	if err != nil {
		w.logError("check file error: %+v", err)
		return false
	}
	for {
		dir, file := filepath.Split(rel)
		if strings.HasPrefix(file, ".") {
			return false
		}
		if dir == "" {
			break
		}
		rel = dir[:len(dir)-1]
	}
	if w.options.ignore != nil {
		return !w.options.ignore.MatchesPath(path) && !w.options.ignore.MatchesPath(path+"/")
	}
	return true
}

func (w *WatchAndRun) shouldWatchFile(path string) bool {
	if strings.HasSuffix(path, "~") {
		return false
	}
	if len(w.options.includeExts) > 0 {
		ext := filepath.Ext(path)
		_, ok := w.options.includeExts[ext]
		if !ok {
			return false
		}
	}
	if w.options.ignore != nil {
		return !w.options.ignore.MatchesPath(path)
	}
	return true
}

func (w *WatchAndRun) handleLoop() {
	defer w.watcher.Close()
	defer w.closeWg.Done()
	for {
		select {
		case <-w.closeCh:
			return
		case e, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.onFsEvent(e)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Fatalf("watcher error %+v", err)
		}
	}
}

func (w *WatchAndRun) onFsEvent(e fsnotify.Event) {
	// 能不能先判断我们对它是否感兴趣, 如果不感兴趣, 就避免调用 os.stat 了. 不过好在 create 事件不会特别多, 性能还好吧.
	if e.Has(fsnotify.Create) {
		// 这里必须用 lstat
		if stat, err := os.Lstat(e.Name); err == nil {
			if stat.IsDir() {
				if w.shouldWatchDir(e.Name) {
					w.addDir(e.Name, true, true)
				}
			} else {
				w.maybeAddFile(e.Name, stat.Mode(), true)
			}
		} // else 大概率是一些临时文件: stat /root/workspace/learn-go2/opensource/fx_test/cri/impl/.impl.go.SoDFiI: no such file or directory
	}
	// 在实践中, write 事件肯定是最多的, 它的处理必须高性能
	if e.Has(fsnotify.Write) {
		if _, ok := w.watched[e.Name]; ok {
			w.logChange("write file %s", e.Name)
			w.notifyRun()
		}
	}
	if e.Has(fsnotify.Remove) || e.Has(fsnotify.Rename) {
		if info, ok := w.watched[e.Name]; ok {
			delete(w.watched, e.Name)
			if info.file {
				w.logChange("remove file %s", e.Name)
				w.notifyRun()
			} else {
				w.logChange("remove dir %s", e.Name)
				dirPath := e.Name + "/"
				for path2, info2 := range w.watched {
					// 有没有更优雅的方式判断 xxx 是 yyy 的子树? 目前我们这里只能遍历
					if strings.HasPrefix(path2, dirPath) {
						delete(w.watched, path2)
						if info2.file {
							w.logChange("unwatch orphan file %s", path2)
							w.notifyRun()
						} else {
							err := w.watcher.Remove(path2)
							w.logChange("unwatch orphan dir %s %+v", path2, err)
						}
					}
				}
			}
		}
	}
}

func (w *WatchAndRun) notifyRun() {
	if w.options.cancelLast {
		w.cancelRun()
	}
	select {
	case w.runCh <- struct{}{}:
	default:
	}
}

func (w *WatchAndRun) cancelRun() {
	cancelDone := make(chan struct{}, 1)
	select {
	case <-w.closeCh:
		return
	case w.cancelRunCh <- cancel{done: cancelDone}:
	}
	select {
	case <-w.closeCh:
		return
	case <-cancelDone:
	}
}

func (w *WatchAndRun) runLoop() {
	defer w.closeWg.Done()
	timer := time.NewTimer(0)
	timer.Stop()
	firstRun := true
	for {
		select {
		case <-w.closeCh:
			return
		case cancelReq := <-w.cancelRunCh:
		drain:
			for {
				select {
				case <-w.runCh:
				default:
					break drain
				}
			}
			timer.Stop()
			cancelReq.done <- struct{}{}
		case <-w.runCh:
			if firstRun {
				timer.Reset(0)
				firstRun = false
			} else {
				timer.Reset(w.options.delay)
			}
		case <-timer.C:
			w.runOnce()
		}
	}
}

func (w *WatchAndRun) runOnce() {
	for _, run := range w.options.run {
		if err := w.runCmd("Run", run); err != nil {
			return
		}
	}
	w.firstRunSuccess = true
}

func (w *WatchAndRun) runCmd(hint string, cmd string) error {
	execCmd := exec.Command("bash", "-c", cmd)
	execCmd.Dir = w.options.root
	execCmd.Env = os.Environ()
	for key, value := range w.options.env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	if w.options.cfgDir != "" {
		execCmd.Env = append(execCmd.Env, "WAR_CFG_DIR="+w.options.cfgDir) //
	}
	if !w.firstRunSuccess {
		execCmd.Env = append(execCmd.Env, "WAR_RUN0=1")
	}
	execCmd.Stdout, execCmd.Stderr = os.Stdout, os.Stderr
	enableProcessGroup(execCmd)
	begin := time.Now()
	if err := execCmd.Start(); err != nil {
		w.logError("%s: start error: %+v", hint, err)
		return err
	}
	w.logSuccess("%s: start pid=%d", hint, execCmd.Process.Pid)
	wait := make(chan error, 1)
	go func() { wait <- execCmd.Wait() }()
	select {
	case cancelReq := <-w.cancelRunCh:
		killBegin := time.Now()
		if err := killCmd(hint, execCmd, wait, w.options.termTimeout); err == nil {
			w.logWarn("%s: cancel run ok, cost=%s", hint, time.Since(killBegin))
		} else {
			w.logError("%s: cancel run error: %+v", hint, err)
		}
		// 再把这个信号扔进去, 让上层去处理
		w.cancelRunCh <- cancelReq
		return errors.New("cancelled")
	case err := <-wait:
		if err != nil {
			w.logError("%s: error %+v", hint, err)
		} else {

			w.logSuccess("%s: done, cost=%s", hint, time.Since(begin))
		}
		return err
	}
}

func (w *WatchAndRun) logChange(format string, args ...any) {
	if w.options.logLevel >= 9 {
		log.Printf(format, args...)
	} else if w.rootWatched && w.options.logLevel >= 1 {
		log.Printf(format, args...)
	}
}

func (w *WatchAndRun) logSuccess(format string, args ...any) {
	log.Println(color.GreenString(format, args...))
}

func (w *WatchAndRun) logWarn(format string, args ...any) {
	log.Println(color.YellowString(format, args...))
}

func (w *WatchAndRun) logError(format string, args ...any) {
	log.Println(color.RedString(format, args...))
}
