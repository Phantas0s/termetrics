package internal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Phantas0s/devdash/internal/platform"
	"github.com/pkg/errors"
)

const (
	// widget config names
	gaBoxRealtime         = "ga.box_real_time"
	gaBoxTotal            = "ga.box_total"
	gaBar                 = "ga.bar"
	gaBarSessions         = "ga.bar_sessions"
	gaBarBounces          = "ga.bar_bounces"
	gaBarUsers            = "ga.bar_users"
	gaBarReturning        = "ga.bar_returning"
	gaBarNewReturning     = "ga.bar_new_returning"
	gaBarPages            = "ga.bar_pages"
	gaBarCountries        = "ga.bar_countries"
	gaBarDevices          = "ga.bar_devices"
	gaTablePages          = "ga.table_pages"
	gaTableTrafficSources = "ga.table_traffic_sources"
	gaTable               = "ga.table"

	// format for every start date / end date
	gaTimeFormat = "2006-01-02"
)

type gaWidget struct {
	tui       *Tui
	analytics *platform.Analytics
	viewID    string
}

// NewGaWidget including all information to connect to the Google Analytics API.
func NewGaWidget(keyfile string, viewID string) (*gaWidget, error) {
	an, err := platform.NewAnalyticsClient(keyfile)
	if err != nil {
		return nil, err
	}

	return &gaWidget{
		analytics: an,
		viewID:    viewID,
	}, nil
}

// CreateWidgets for Google Analytics.
func (g *gaWidget) CreateWidgets(widget Widget, tui *Tui) (f func() error, err error) {
	g.tui = tui

	switch widget.Name {
	case gaBoxRealtime:
		f, err = g.realTimeUser(widget)
	case gaBoxTotal:
		f, err = g.totalMetric(widget)
	case gaBarSessions:
		f, err = g.barMetric(widget, platform.XHeaderTime)
	case gaBarUsers:
		f, err = g.users(widget)
	case gaBar:
		f, err = g.barMetric(widget, platform.XHeaderTime)
	case gaTablePages:
		f, err = g.table(widget, "Page")
	case gaTableTrafficSources:
		f, err = g.trafficSource(widget)
	case gaBarNewReturning:
		f, err = g.stackedBarNewReturningUsers(widget)
	case gaBarDevices:
		f, err = g.stackedBarDevices(widget)
	case gaBarReturning:
		f, err = g.barReturning(widget)
	case gaBarPages:
		f, err = g.barPages(widget)
	case gaBarCountries:
		f, err = g.barCountries(widget)
	case gaBarBounces:
		f, err = g.barBounces(widget)
	case gaTable:
		f, err = g.table(widget, widget.Options[optionDimension])
	default:
		return nil, errors.Errorf("can't find the widget %s", widget.Name)
	}

	return
}

func (g *gaWidget) totalMetric(widget Widget) (f func() error, err error) {
	startDate, endDate, err := ExtractTimeRange(time.Now(), widget.Options)
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("Total %s from %s to %s", ExtractMetric(widget.Options), startDate.Format(gaTimeFormat), endDate.Format(gaTimeFormat))
	if _, ok := widget.Options[optionTitle]; ok {
		title = widget.Options[optionTitle]
	}

	global := false
	if _, ok := widget.Options[optionGlobal]; ok {
		global, err = strconv.ParseBool(widget.Options[optionGlobal])
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse string %s to bool", widget.Options[optionGlobal])
		}
	}

	users, err := g.analytics.SimpleMetric(
		platform.AnalyticValues{
			ViewID:    g.viewID,
			StartDate: startDate.Format(gaTimeFormat),
			EndDate:   endDate.Format(gaTimeFormat),
			Global:    global,
			Metrics:   []string{ExtractMetric(widget.Options)},
		},
	)
	if err != nil {
		return nil, err
	}

	f = func() error {
		return g.tui.AddTextBox(users, title, widget.Options)
	}

	return
}

// GaRTActiveUser get the real time active users from Google Analytics
func (g *gaWidget) realTimeUser(widget Widget) (f func() error, err error) {
	title := " Real time users "
	if _, ok := widget.Options[optionTitle]; ok {
		title = widget.Options[optionTitle]
	}

	users, err := g.analytics.RealTimeUsers(g.viewID)
	if err != nil {
		return nil, err
	}

	f = func() error {
		return g.tui.AddTextBox(
			users,
			title,
			widget.Options,
		)
	}

	return
}

func (g *gaWidget) users(widget Widget) (f func() error, err error) {
	if widget.Options == nil {
		widget.Options = map[string]string{}
	}

	widget.Options[optionMetric] = "users"
	xHeader := platform.XHeaderTime

	return g.barMetric(widget, xHeader)
}

func (g *gaWidget) barReturning(widget Widget) (f func() error, err error) {
	if widget.Options == nil {
		widget.Options = map[string]string{}
	}

	widget.Options[optionMetric] = "users"
	widget.Options[optionDimensions] = "user_type"
	widget.Options[optionTitle] = " Returning users "

	return g.barMetric(widget, platform.XHeaderTime)
}

func (g *gaWidget) barPages(widget Widget) (f func() error, err error) {
	if widget.Options == nil {
		widget.Options = map[string]string{}
	}
	widget.Options[optionDimensions] = "page_path"
	widget.Options[optionMetric] = "page_views"

	if _, ok := widget.Options[optionFilters]; !ok {
		return nil, errors.New("The widget ga.bar_pages require a filter (relative url of your page, i.e '/my-super-page/')")
	}

	if _, ok := widget.Options[optionTitle]; !ok {
		widget.Options[optionTitle] = widget.Options[optionFilters]
	}

	return g.barMetric(widget, platform.XHeaderTime)
}

func (g *gaWidget) barCountries(widget Widget) (f func() error, err error) {
	if widget.Options == nil {
		widget.Options = map[string]string{}
	}
	widget.Options[optionDimensions] = "country"
	widget.Options[optionMetric] = "sessions"

	if _, ok := widget.Options[optionTitle]; !ok {
		widget.Options[optionTitle] = widget.Options[optionFilters]
	}

	return g.barMetric(widget, platform.XHeaderOtherDim)
}

func (g *gaWidget) barBounces(widget Widget) (f func() error, err error) {
	if widget.Options == nil {
		widget.Options = map[string]string{}
	}
	widget.Options[optionMetric] = "bounces"
	widget.Options[optionTitle] += " Bounces "

	return g.barMetric(widget, platform.XHeaderTime)
}

func (g *gaWidget) barMetric(widget Widget, xHeader uint16) (f func() error, err error) {
	global := false
	if _, ok := widget.Options[optionGlobal]; ok {
		global, err = strconv.ParseBool(widget.Options[optionGlobal])
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse string %s to bool", widget.Options[optionGlobal])
		}
	}

	startDate, endDate, err := ExtractTimeRange(time.Now(), widget.Options)
	if err != nil {
		return nil, err
	}

	filters := []string{}
	if _, ok := widget.Options[optionFilters]; ok {
		if len(widget.Options[optionFilters]) > 0 {
			filters = strings.Split(strings.TrimSpace(widget.Options[optionFilters]), ",")
		}
	}

	timePeriod := "day"
	if _, ok := widget.Options[optionTimePeriod]; ok {
		timePeriod = strings.TrimSpace(widget.Options[optionTimePeriod])
	}

	title := fmt.Sprintf(" %s per %s ", strings.Title(ExtractMetric(widget.Options)), timePeriod)
	if _, ok := widget.Options[optionTitle]; ok {
		title = widget.Options[optionTitle]
	}

	dim, val, err := g.analytics.BarMetric(
		platform.AnalyticValues{
			ViewID:     g.viewID,
			StartDate:  startDate.Format(gaTimeFormat),
			EndDate:    endDate.Format(gaTimeFormat),
			TimePeriod: timePeriod,
			Global:     global,
			Metrics:    []string{ExtractMetric(widget.Options)},
			Dimensions: ExtractDimensions(widget.Options),
			Filters:    filters,
			XHeaders:   xHeader,
		},
	)
	if err != nil {
		return nil, err
	}

	f = func() error {
		return g.tui.AddBarChart(val, dim, title, widget.Options)
	}

	return f, nil
}

func (g *gaWidget) table(widget Widget, firstHeader string) (f func() error, err error) {
	global := false
	if _, ok := widget.Options[optionGlobal]; ok {
		global, err = strconv.ParseBool(widget.Options[optionGlobal])
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse string %s to bool", widget.Options[optionGlobal])
		}
	}

	dimension := "page_path"
	if _, ok := widget.Options[optionDimension]; ok {
		// TODO is the condition useful?
		if len(widget.Options[optionDimension]) > 0 {
			dimension = widget.Options[optionDimension]
		}
	}

	metrics := []string{"sessions", "page_views", "entrances", "unique_page_views"}
	if _, ok := widget.Options[optionMetrics]; ok {
		if len(widget.Options[optionMetrics]) > 0 {
			metrics = strings.Split(strings.TrimSpace(widget.Options[optionMetrics]), ",")
		}
	}

	orders := []string{metrics[0] + " desc"}
	if _, ok := widget.Options[optionOrder]; ok {
		if len(widget.Options[optionOrder]) > 0 {
			orders = strings.Split(strings.TrimSpace(widget.Options[optionOrder]), ",")
		}
	}

	startDate, endDate, err := ExtractTimeRange(time.Now(), widget.Options)
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("%s from %s to %s", firstHeader, startDate.Format(gaTimeFormat), endDate.Format(gaTimeFormat))
	if _, ok := widget.Options[optionTitle]; ok {
		title = widget.Options[optionTitle]
	}

	filters := []string{}
	if _, ok := widget.Options[optionFilters]; ok {
		if len(widget.Options[optionFilters]) > 0 {
			filters = strings.Split(strings.TrimSpace(widget.Options[optionFilters]), ",")
		}
	}

	var rowLimit int64 = 5
	if _, ok := widget.Options[optionRowLimit]; ok {
		rowLimit, err = strconv.ParseInt(widget.Options[optionRowLimit], 0, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "%s must be a number", widget.Options[optionRowLimit])
		}
	}

	headers, dim, val, err := g.analytics.Table(
		platform.AnalyticValues{
			ViewID:     g.viewID,
			StartDate:  startDate.Format(gaTimeFormat),
			EndDate:    endDate.Format(gaTimeFormat),
			Global:     global,
			Metrics:    metrics,
			Dimensions: []string{dimension},
			Filters:    filters,
			Orders:     orders,
			RowLimit:   rowLimit,
		},
		firstHeader,
	)
	if err != nil {
		return nil, err
	}

	if int(rowLimit) > len(dim) {
		rowLimit = int64(len(dim))
	}

	var charLimit int64 = 20
	if _, ok := widget.Options[optionCharLimit]; ok {
		charLimit, err = strconv.ParseInt(widget.Options[optionCharLimit], 0, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "%s must be a number", widget.Options[optionCharLimit])
		}
	}

	finalTable := formatTable(rowLimit, dim, val, charLimit, headers)

	f = func() error {
		return g.tui.AddTable(finalTable, title, widget.Options)
	}

	return
}

func formatTable(
	rowLimit int64,
	dim []string,
	val [][]string,
	charLimit int64,
	headers []string,
) [][]string {
	table := [][]string{headers}

	for k, v := range val {
		if k == int(rowLimit) {
			break
		}

		p := strings.Trim(dim[k], " ")
		if len(p) > int(charLimit) {
			p = p[:charLimit]
		}

		// Add dimension header
		row := []string{p}
		row = append(row, v...)
		table = append(table, row)
	}

	return table
}

func (g *gaWidget) trafficSource(widget Widget) (f func() error, err error) {
	if widget.Options == nil {
		widget.Options = map[string]string{}
	}

	widget.Options[optionDimension] = "traffic_source"

	return g.table(widget, "Source")
}

func (g *gaWidget) stackedBarNewReturningUsers(widget Widget) (func() error, error) {
	if widget.Options == nil {
		widget.Options = map[string]string{}
	}

	widget.Options[optionDimensions] = "user_type"

	return g.stackedBar(widget)
}

func (g *gaWidget) stackedBarDevices(widget Widget) (f func() error, err error) {
	if widget.Options == nil {
		widget.Options = map[string]string{}
	}

	widget.Options[optionDimensions] = "device_category"

	return g.stackedBar(widget)
}

func (g *gaWidget) stackedBar(widget Widget) (f func() error, err error) {
	// defaults
	startDate, endDate, err := ExtractTimeRange(time.Now(), widget.Options)
	if err != nil {
		return nil, err
	}

	timePeriod := "day"
	if _, ok := widget.Options[optionTimePeriod]; ok {
		timePeriod = strings.TrimSpace(widget.Options[optionTimePeriod])
	}

	dim, val, err := g.analytics.StackedBar(
		platform.AnalyticValues{
			ViewID:     g.viewID,
			StartDate:  startDate.Format(gaTimeFormat),
			EndDate:    endDate.Format(gaTimeFormat),
			TimePeriod: timePeriod,
			Metrics:    []string{ExtractMetric(widget.Options)},
			Dimensions: ExtractDimensions(widget.Options),
		},
	)
	if err != nil {
		return nil, err
	}

	// Only support 5 different colors for now
	colors := []uint16{blue, green, yellow, red, magenta}
	if _, ok := widget.Options[optionFirstColor]; ok {
		colors[0] = colorLookUp[widget.Options[optionFirstColor]]
	}
	if _, ok := widget.Options[optionSecondColor]; ok {
		colors[1] = colorLookUp[widget.Options[optionSecondColor]]
	}
	if _, ok := widget.Options[optionThirdColor]; ok {
		colors[2] = colorLookUp[widget.Options[optionThirdColor]]
	}
	if _, ok := widget.Options[optionFourthColor]; ok {
		colors[3] = colorLookUp[widget.Options[optionFourthColor]]
	}
	if _, ok := widget.Options[optionFifthColor]; ok {
		colors[4] = colorLookUp[widget.Options[optionFifthColor]]
	}

	var data [8][]int
	title := fmt.Sprintf(strings.Trim(strings.Title(ExtractMetric(widget.Options)), "_")) + " - "
	count := 0
	for k, v := range val {
		data[count] = v

		if count != 0 {
			title += "/ "
		}
		title += fmt.Sprintf("%s (%s) ", k, colorStr(colors[count]))

		count++
	}
	if _, ok := widget.Options[optionTitle]; ok {
		title = widget.Options[optionTitle]
	}

	f = func() error {
		return g.tui.AddStackedBarChart(data, dim, title, colors, widget.Options)
	}

	return
}

func ExtractTimeRange(base time.Time, widgetOptions map[string]string) (sd time.Time, ed time.Time, err error) {
	startDate := "7_days_ago"
	if _, ok := widgetOptions[optionStartDate]; ok {
		startDate = widgetOptions[optionStartDate]
	}

	endDate := "today"
	if _, ok := widgetOptions[optionEndDate]; ok {
		endDate = widgetOptions[optionEndDate]
	}

	sd, ed, err = platform.ConvertDates(base, startDate, endDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return
}

func ExtractDimensions(widgetOptions map[string]string) (dimensions []string) {
	dimensions = []string{}
	if _, ok := widgetOptions[optionDimensions]; ok {
		if len(widgetOptions[optionDimensions]) > 0 {
			dimensions = strings.Split(strings.TrimSpace(widgetOptions[optionDimensions]), ",")
		}
	}

	return
}

func ExtractMetric(widgetOptions map[string]string) (metric string) {
	metric = "sessions"
	if _, ok := widgetOptions[optionMetric]; ok {
		metric = widgetOptions[optionMetric]
	}

	return
}
