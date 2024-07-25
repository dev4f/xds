package observer

import (
	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
	"xDS/internal/constant"
)

type OperationType int

const (
	Create OperationType = iota
	Remove
	Modify
)

type NotifyMessage struct {
	Operation OperationType
	FilePath  string
}

func (n NotifyMessage) OperationName() string {
	switch n.Operation {
	case Create:
		return "create"
	case Remove:
		return "remove"
	case Modify:
		return "modify"
	}
	return "unknown"
}

func (n NotifyMessage) IsNotSupported() bool {
	return !n.IsLds() && !n.IsRds() && !n.IsRls() && !n.IsEds()
}

func (n NotifyMessage) IsLds() bool {
	return strings.HasSuffix(n.FilePath, constant.ListenerFileSuffix)
}

func (n NotifyMessage) IsRds() bool {
	return strings.HasSuffix(n.FilePath, constant.RouteFileSuffix)
}

func (n NotifyMessage) IsRls() bool {
	return strings.HasSuffix(n.FilePath, constant.RatelimitFileSuffix)
}

func (n NotifyMessage) IsEds() bool {
	return strings.HasSuffix(n.FilePath, constant.EndpointFileSuffix)
}

func (n NotifyMessage) IsCds() bool {
	return strings.HasSuffix(n.FilePath, "cds.yaml")
}

func Watch(directory string, notifyCh chan<- NotifyMessage) {

	w := watcher.New()
	w.SetMaxEvents(1)
	w.FilterOps(
		watcher.Create,
		watcher.Write,
		watcher.Rename,
		watcher.Move,
	)

	// Only files that match the regular expression during file listings
	// will be watched.
	r := regexp.MustCompile(`^.*\.yaml$`)
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

	defer w.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-w.Event:
				notifyCh <- NotifyMessage{
					Operation: Modify,
					FilePath:  event.Path,
				}
			case err := <-w.Error:
				log.Errorf("watcher error: %v\n", err)
			case <-w.Closed:
				return
			}
			//select {
			//case event, ok := <-watcher.Events:
			//	if !ok {
			//		return
			//	}
			//	if event.Op&fsnotify.Write == fsnotify.Write {
			//		notifyCh <- NotifyMessage{
			//			Operation: Modify,
			//			FilePath:  event.Name,
			//		}
			//	} else if event.Op&fsnotify.Create == fsnotify.Create {
			//		notifyCh <- NotifyMessage{
			//			Operation: Create,
			//			FilePath:  event.Name,
			//		}
			//	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
			//		notifyCh <- NotifyMessage{
			//			Operation: Remove,
			//			FilePath:  event.Name,
			//		}
			//	}
			//
			//case err, ok := <-watcher.Errors:
			//	if !ok {
			//		return
			//	}
			//	log.Println("error:", err)
			//}
		}
	}()

	err := w.Add(directory)
	if err != nil {
		log.Fatal(err)
	}

	<-done
}
