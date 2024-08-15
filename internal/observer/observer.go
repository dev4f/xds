package observer

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
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
	return !n.IsLds() && !n.IsCds() && !n.IsRds() && !n.IsRls() && !n.IsEds()
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
	return strings.HasSuffix(n.FilePath, constant.ClusterFileSuffix)
}

func Watch(directory string, notifyCh chan<- NotifyMessage) {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		var (
			timer  *time.Timer
			events = make(map[string]fsnotify.Event)
		)
		timer = time.NewTimer(time.Millisecond)
		<-timer.C // timer should be expired at first
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Infof("event: %v", event)
				// override event for the same file
				eventId := event.Name + event.Op.String()
				events[eventId] = event
				timer.Reset(time.Millisecond * 100)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error(err)

			case <-timer.C:

				log.Infof("events: %v", events)
				for _, e := range events {
					notifyCh <- eventToMessage(e)
				}
				events = make(map[string]fsnotify.Event)
			}
		}
	}()

	err = watcher.Add(directory)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func eventToMessage(e fsnotify.Event) NotifyMessage {
	if e.Op&fsnotify.Write == fsnotify.Write {
		return NotifyMessage{
			Operation: Modify,
			FilePath:  e.Name,
		}
	} else if e.Op&fsnotify.Create == fsnotify.Create {
		return NotifyMessage{
			Operation: Create,
			FilePath:  e.Name,
		}
	} else if e.Op&fsnotify.Remove == fsnotify.Remove || e.Op&fsnotify.Rename == fsnotify.Rename {
		return NotifyMessage{
			Operation: Remove,
			FilePath:  e.Name,
		}
	}
	return NotifyMessage{}
}
