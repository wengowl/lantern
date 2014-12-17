package statreporter

import (
	"fmt"
	"testing"
	"time"

	"github.com/getlantern/testify/assert"
)

func TestAll(t *testing.T) {
	instanceid := "testinstance"

	// Set up fake statshub
	reportCh := make(chan report)

	// Set up two dim groups with a dim in common and a dim different
	dg1 := Dim("a", "1").And("b", "1")
	dg2 := Dim("b", "2").And("a", "1")

	// Start reporting
	err := doConfigure(&Config{
		ReportingPeriod: 100 * time.Millisecond,
		InstanceId:      instanceid,
	}, func(r report) error {
		go func() {
			reportCh <- r
		}()
		return nil
	})
	if err != nil {
		t.Fatalf("Unable to configure statreporter: %s", err)
	}

	// Add stats
	dg1.Increment("incra").Add(1)
	dg1.Increment("incra").Add(1)
	dg1.Increment("incrb").Set(1)
	dg1.Increment("incrb").Set(25)
	dg1.Gauge("gaugea").Add(2)
	dg1.Gauge("gaugea").Add(2)
	dg1.Gauge("gaugeb").Set(2)
	dg1.Gauge("gaugeb").Set(48)

	originalReporter := currentReporter

	// Reconfigure reporting
	doConfigure(&Config{
		ReportingPeriod: 200 * time.Millisecond,
		InstanceId:      instanceid,
	}, func(r report) error {
		go func() {
			reportCh <- r
		}()
		return nil
	})

	// Get the first report
	report1 := <-reportCh

	dg2.Increment("incra").Add(1)
	dg2.Increment("incra").Add(1)
	dg2.Increment("incrb").Set(1)
	dg2.Increment("incrb").Set(25)
	dg2.Gauge("gaugea").Add(2)
	dg2.Gauge("gaugea").Add(2)
	dg2.Gauge("gaugeb").Set(2)
	dg2.Gauge("gaugeb").Set(48)

	updatedReporter := currentReporter

	assert.NotEqual(t, originalReporter, updatedReporter, "Reporter should have changed after reconfiguring")

	expectedReport1 := report{
		"dims": map[string]string{
			"a":       "1",
			"b":       "1",
			"country": "us",
		},
		"increments": stats{
			"incra": 2,
			"incrb": 25,
		},
		"gauges": stats{
			"gaugea": 4,
			"gaugeb": 48,
		},
	}
	expectedReport2 := report{
		"dims": map[string]string{
			"a":       "1",
			"b":       "2",
			"country": "cn",
		},
		"increments": stats{
			"incra": 2,
			"incrb": 25,
		},
		"gauges": stats{
			"gaugea": 4,
			"gaugeb": 48,
		},
	}

	// Get the 2nd report
	report2 := <-reportCh

	// Since reports can be made in unpredictable order, figure out which one
	// is which
	if report1["dims"].(map[string]string)["b"] == "2" {
		// switch
		report1, report2 = report2, report1
	}

	compareReports(t, expectedReport1, report1, "1st")
	compareReports(t, expectedReport2, report2, "2nd")
}

func compareReports(t *testing.T, expected report, actual report, index string) {
	expectedDims := expected["dims"].(map[string]string)
	actualDims := actual["dims"].(map[string]string)

	assert.Equal(t, expectedDims["a"], actualDims["a"], fmt.Sprintf("On %s, dim a should match", index))
	assert.Equal(t, expectedDims["b"], actualDims["b"], fmt.Sprintf("On %s, dim b should match", index))

	assert.Equal(t, expected["increments"], actual["increments"], fmt.Sprintf("On %s, increments should match", index))
	assert.Equal(t, expected["gauges"], actual["gauges"], fmt.Sprintf("On %s, gauges should match", index))
}
