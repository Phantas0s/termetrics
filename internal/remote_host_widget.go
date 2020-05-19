package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/Phantas0s/devdash/internal/platform"
	"github.com/pkg/errors"
)

const (
	rhUptime = "rh.box_uptime"
	rhMemory = "rh.bar_memory"
)

type remoteHostWidget struct {
	tui     *Tui
	service *platform.RemoteHost
}

func NewRemoteHostWidget(username, addr string) (*remoteHostWidget, error) {
	service, err := platform.NewRemoteHost(username, addr)
	if err != nil {
		return nil, err
	}

	return &remoteHostWidget{
		service: service,
	}, nil
}

func (ms *remoteHostWidget) CreateWidgets(widget Widget, tui *Tui) (f func() error, err error) {
	ms.tui = tui

	switch widget.Name {
	case rhUptime:
		f, err = ms.boxUptime(widget)
	case rhMemory:
		f, err = ms.barGetMemory(widget)
	default:
		return nil, errors.Errorf("can't find the widget %s", widget.Name)
	}

	return
}

func (ms *remoteHostWidget) boxUptime(widget Widget) (f func() error, err error) {
	title := "Uptime"
	if _, ok := widget.Options[optionTitle]; ok {
		title = widget.Options[optionTitle]
	}

	uptime, err := ms.service.Uptime()
	if err != nil {
		return nil, err
	}

	f = func() error {
		return ms.tui.AddTextBox(formatSeconds(time.Duration(uptime)), title, widget.Options)
	}

	return
}

func formatSeconds(dur time.Duration) string {
	dur = dur - (dur % time.Second)
	var days int
	for dur.Hours() > 24.0 {
		days++
		dur -= 24 * time.Hour
	}
	for dur.Hours() > 24.0 {
		days++
		dur -= 24 * time.Hour
	}

	s1 := dur.String()
	s2 := ""
	if days > 0 {
		s2 = fmt.Sprintf("%dd ", days)
	}
	for _, ch := range s1 {
		s2 += string(ch)
		if ch == 'h' || ch == 'm' {
			s2 += " "
		}
	}
	return s2
}

func (ms *remoteHostWidget) barGetMemory(widget Widget) (f func() error, err error) {
	title := "Memory"
	if _, ok := widget.Options[optionTitle]; ok {
		title = widget.Options[optionTitle]
	}

	metrics := []string{"MemTotal", "MemFree", "MemAvailable"}
	if _, ok := widget.Options[optionMetrics]; ok {
		if len(widget.Options[optionMetrics]) > 0 {
			metrics = strings.Split(strings.TrimSpace(widget.Options[optionMetrics]), ",")
		}
	}

	unit := "kb"
	if _, ok := widget.Options[optionUnit]; ok {
		unit = widget.Options[optionUnit]
	}

	fmt.Println(widget.Options)
	mem, err := ms.service.Memory(metrics, unit)
	if err != nil {
		return nil, err
	}

	f = func() error {
		return ms.tui.AddBarChart(mem, metrics, title, widget.Options)
	}

	return
}

// func (ms *monitorServerWidget) table(widget Widget, firstHeader string) (f func() error, err error) {
// }
