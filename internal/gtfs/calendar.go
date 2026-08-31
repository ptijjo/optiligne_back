package gtfs

import "time"

// ServiceActive indique si service_id circule à date (calendrier + exceptions).
func ServiceActive(cals []Calendar, dates []CalendarDate, serviceID string, day time.Time) bool {
	ymd := day.Format("20060102")
	active := false
	for _, c := range cals {
		if c.ServiceID != serviceID {
			continue
		}
		if ymd < c.StartDate || ymd > c.EndDate {
			continue
		}
		if weekdayOn(c, day.Weekday()) {
			active = true
		}
	}
	for _, d := range dates {
		if d.ServiceID != serviceID || d.Date != ymd {
			continue
		}
		switch d.ExceptionType {
		case 1:
			active = true
		case 2:
			active = false
		}
	}
	return active
}

func weekdayOn(c Calendar, w time.Weekday) bool {
	switch w {
	case time.Monday:
		return c.Monday
	case time.Tuesday:
		return c.Tuesday
	case time.Wednesday:
		return c.Wednesday
	case time.Thursday:
		return c.Thursday
	case time.Friday:
		return c.Friday
	case time.Saturday:
		return c.Saturday
	case time.Sunday:
		return c.Sunday
	default:
		return false
	}
}
