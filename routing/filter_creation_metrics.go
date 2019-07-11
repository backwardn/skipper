package routing

import (
	"time"

	"github.com/zalando/skipper/filters"
)

const (
	maxAge              = 2
	metricsPrefix       = "routeCreationTime."
	defaultCreationTime = "default"
)

type FilterCreationMetrics struct {
	metrics   filters.Metrics
	configIds map[string]map[string]int
}

func NewFilterCreationMetrics(metrics filters.Metrics) *FilterCreationMetrics {
	return &FilterCreationMetrics{metrics: metrics, configIds: map[string]map[string]int{}}
}

// Do implements routing.PostProcessor and records the filter creation time.
func (m *FilterCreationMetrics) Do(routes []*Route) []*Route {
	for _, r := range routes {
		for name, start := range m.startTimes(r) {
			m.metrics.MeasureSince(metricsPrefix+name, start)
		}
	}

	m.pruneCache()

	return routes
}

func (m *FilterCreationMetrics) startTimes(route *Route) map[string]time.Time {
	startTimes := map[string]time.Time{}

	t := m.configInfoStartTime(route.ConfigInfo, defaultCreationTime)
	if !t.IsZero() {
		startTimes[defaultCreationTime] = t
	}

	for _, f := range route.Filters {
		t := m.filterStartTime(f)
		o, exists := startTimes[f.Name]

		if t.IsZero() {
			continue
		}

		if !exists || t.Before(o) {
			startTimes[f.Name] = t
		}
	}

	return startTimes
}

func (m *FilterCreationMetrics) filterStartTime(filter *RouteFilter) time.Time {
	if info, ok := filter.Filter.(filters.ConfigInfo); ok {
		return m.configInfoStartTime(info, filter.Name)
	}

	return time.Time{}
}

func (m *FilterCreationMetrics) configInfoStartTime(info filters.ConfigInfo, name string) time.Time {
	if info == nil {
		return time.Time{}
	}

	id := info.ConfigID()
	created := info.ConfigCreated()
	if created.IsZero() || id == "" {
		return time.Time{}
	}
	filterCache := m.configIds[name]
	if filterCache == nil {
		filterCache = map[string]int{}
		m.configIds[name] = filterCache
	}
	_, exists := filterCache[id]
	filterCache[id] = 0
	if !exists {
		return created
	}
	return time.Time{}
}

func (m *FilterCreationMetrics) pruneCache() {
	for _, fc := range m.configIds {
		for id, age := range fc {
			age++
			if age > maxAge {
				delete(fc, id)
			} else {
				fc[id] = age
			}
		}
	}
}
