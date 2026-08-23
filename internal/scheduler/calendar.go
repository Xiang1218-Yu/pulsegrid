package scheduler

import (
	"sort"
	"sync"
	"time"
)

type Window struct {
	Weekday     time.Weekday `json:"weekday"`
	StartMinute int          `json:"start_minute"`
	EndMinute   int          `json:"end_minute"`
}

type Calendar struct {
	mu       sync.RWMutex
	location *time.Location
	windows  []Window
	holidays map[string]bool
}

type Slot struct {
	At        time.Time `json:"at"`
	Available bool      `json:"available"`
	Reason    string    `json:"reason,omitempty"`
}

type Schedule struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Timezone string   `json:"timezone"`
	Windows  []Window `json:"windows"`
	Holidays []string `json:"holidays,omitempty"`
}

func NewCalendar(location *time.Location) *Calendar {
	if location == nil {
		location = time.UTC
	}
	return &Calendar{location: location, windows: []Window{}, holidays: map[string]bool{}}
}

func (c *Calendar) AddWindow(window Window) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if window.StartMinute < 0 {
		window.StartMinute = 0
	}
	if window.EndMinute > 1440 {
		window.EndMinute = 1440
	}
	if window.EndMinute <= window.StartMinute {
		return
	}
	c.windows = append(c.windows, window)
	sort.Slice(c.windows, func(i, j int) bool {
		if c.windows[i].Weekday != c.windows[j].Weekday {
			return c.windows[i].Weekday < c.windows[j].Weekday
		}
		return c.windows[i].StartMinute < c.windows[j].StartMinute
	})
}

func (c *Calendar) AddHoliday(date time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.holidays[date.In(c.location).Format("2006-01-02")] = true
}

func (c *Calendar) IsAvailable(at time.Time) Slot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	local := at.In(c.location)
	date := local.Format("2006-01-02")
	if c.holidays[date] {
		return Slot{At: at, Available: false, Reason: "holiday"}
	}
	minute := local.Hour()*60 + local.Minute()
	for _, window := range c.windows {
		if window.Weekday == local.Weekday() && minute >= window.StartMinute && minute < window.EndMinute {
			return Slot{At: at, Available: true}
		}
	}
	return Slot{At: at, Available: false, Reason: "outside sending window"}
}

func (c *Calendar) NextAvailable(from time.Time, horizon time.Duration) Slot {
	if horizon <= 0 {
		horizon = 7 * 24 * time.Hour
	}
	step := time.Minute
	for elapsed := time.Duration(0); elapsed <= horizon; elapsed += step {
		at := from.Add(elapsed)
		slot := c.IsAvailable(at)
		if slot.Available {
			return slot
		}
	}
	return Slot{At: from.Add(horizon), Available: false, Reason: "no available slot"}
}

func (c *Calendar) Schedule(value Schedule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.windows = append([]Window(nil), value.Windows...)
	c.holidays = map[string]bool{}
	for _, holiday := range value.Holidays {
		c.holidays[holiday] = true
	}
}

func (c *Calendar) Snapshot() Schedule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	holidays := make([]string, 0, len(c.holidays))
	for holiday := range c.holidays {
		holidays = append(holidays, holiday)
	}
	sort.Strings(holidays)
	windows := append([]Window(nil), c.windows...)
	return Schedule{Timezone: c.location.String(), Windows: windows, Holidays: holidays}
}

func (c *Calendar) IsBusinessDay(at time.Time) bool {
	weekday := at.In(c.location).Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.holidays[at.In(c.location).Format("2006-01-02")]
}

func (c *Calendar) CountAvailableSlots(from time.Time, days int) int {
	if days < 1 {
		days = 1
	}
	count := 0
	for index := 0; index < days; index++ {
		day := from.AddDate(0, 0, index)
		for minute := 0; minute < 1440; minute += 30 {
			if c.IsAvailable(day.Add(time.Duration(minute) * time.Minute)).Available {
				count++
			}
		}
	}
	return count
}

func DefaultBusinessCalendar() *Calendar {
	calendar := NewCalendar(time.UTC)
	for _, weekday := range []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday} {
		calendar.AddWindow(Window{Weekday: weekday, StartMinute: 9 * 60, EndMinute: 17 * 60})
	}
	return calendar
}
