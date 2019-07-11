package routing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zalando/skipper/eskip"
	"github.com/zalando/skipper/filters"
	"github.com/zalando/skipper/filters/filtertest"
	"github.com/zalando/skipper/metrics/metricstest"
)

var time0 = time.Now()
var time1 = time.Now().Add(1)

type configInfoFilter struct {
	id     string
	crated time.Time
}

func (c configInfoFilter) Request(filters.FilterContext) {
	panic("not supported")
}

func (c configInfoFilter) Response(filters.FilterContext) {
	panic("not supported")
}

func (c configInfoFilter) ConfigID() string {
	return c.id
}

func (c configInfoFilter) ConfigCreated() time.Time {
	return c.crated
}

func TestFilterCreationMetrics_Do(t *testing.T) {
	for _, tt := range []struct {
		name            string
		route           eskip.Route
		expectedMetrics []string
	}{
		{
			name:  "no start time provided",
			route: eskip.Route{},
		},
		{
			name:            "start time provided",
			route:           eskip.Route{ConfigInfo: configInfoFilter{"config1", time0}},
			expectedMetrics: []string{"routeCreationTime.default"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			metrics := metricstest.MockMetrics{}
			NewFilterCreationMetrics(&metrics).Do([]*Route{{Route: tt.route}})

			metrics.WithMeasures(func(measures map[string][]time.Duration) {
				assert.Len(t, measures, len(tt.expectedMetrics))

				for _, e := range tt.expectedMetrics {
					assert.Containsf(t, measures, e, "measure metrics do not contain %q", e)
				}
			})
		})
	}
}

func TestFilterCreationMetrics_startTimes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		route    Route
		expected map[string]time.Time
	}{
		{
			name:     "no start time provided",
			route:    Route{},
			expected: map[string]time.Time{},
		},
		{
			name:     "start time provided",
			route:    Route{Route: eskip.Route{ConfigInfo: configInfoFilter{"config1", time0}}},
			expected: map[string]time.Time{"default": time0},
		},
		{
			name: "start time from filter",
			route: Route{Filters: []*RouteFilter{
				{
					Name:   "filter",
					Filter: configInfoFilter{"config0", time0},
				},
				{
					Name:   "filter",
					Filter: configInfoFilter{"config1", time1},
				},
			}},
			expected: map[string]time.Time{"filter": time0},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			metrics := metricstest.MockMetrics{}
			s := NewFilterCreationMetrics(&metrics).startTimes(&tt.route)

			assert.Equal(t, tt.expected, s)
		})
	}
}

func TestFilterCreationMetrics_pruneCache(t *testing.T) {
	for _, tt := range []struct {
		name              string
		configIds         map[string]map[string]int
		expectedConfigIds map[string]map[string]int
	}{
		{
			name:              "age increased",
			configIds:         map[string]map[string]int{"filter": {"config0": 0, "config1": 1}},
			expectedConfigIds: map[string]map[string]int{"filter": {"config0": 1, "config1": 2}},
		},
		{
			name:              "entry pruned",
			configIds:         map[string]map[string]int{"filter": {"config0": 0, "config1": maxAge}},
			expectedConfigIds: map[string]map[string]int{"filter": {"config0": 1}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &FilterCreationMetrics{
				configIds: tt.configIds,
			}
			m.pruneCache()
			assert.Equal(t, tt.expectedConfigIds, m.configIds)
		})
	}
}

func TestFilterCreationMetrics_filterStartTime(t *testing.T) {
	for _, tt := range []struct {
		name      string
		configIds map[string]map[string]int
		filter    filters.Filter
		expected  time.Time
	}{
		{
			name:      "not config info",
			configIds: map[string]map[string]int{},
			filter:    &filtertest.Filter{},
			expected:  time.Time{},
		},
		{
			name:      "config info with no time",
			configIds: map[string]map[string]int{},
			filter:    configInfoFilter{id: "config1"},
			expected:  time.Time{},
		},
		{
			name:      "no config exists",
			configIds: map[string]map[string]int{},
			filter:    configInfoFilter{"config1", time0},
			expected:  time0,
		},
		{
			name:      "same config",
			configIds: map[string]map[string]int{"filter": {"config0": 0}},
			filter:    configInfoFilter{"config0", time0},
			expected:  time.Time{},
		},
		{
			name:      "new config",
			configIds: map[string]map[string]int{"filter": {"config0": 0}},
			filter:    configInfoFilter{"config1", time1},
			expected:  time1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &FilterCreationMetrics{
				configIds: tt.configIds,
			}
			assert.Equal(t, tt.expected, m.filterStartTime(&RouteFilter{
				Filter: tt.filter,
				Name:   "filter",
			}))
		})
	}
}
