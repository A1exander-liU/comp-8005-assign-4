package controller

import (
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
)

func (c *Controller) printJobResults(result string, err error, chunkID int) {
	startPassword := shared.EncodeBase(c.chunks[chunkID].start, shared.SearchSpace)
	endPassword := shared.EncodeBase(c.chunks[chunkID].end, shared.SearchSpace)
	jobMetric, _ := c.metric.GetJobMetric(chunkID)

	resultTitle := fmt.Sprintf("==== CHUNK: '%s' to '%s' RESULTS (seconds)", startPassword, endPassword)

	var passwordTitle string
	if err != nil {
		passwordTitle = fmt.Sprintf("PASSWORD NOT FOUND: %s", err)
	} else {
		passwordTitle = fmt.Sprintf("PASSWORD: %s", result)
	}

	dispatchTime := jobMetric.dispatchTime
	assignTime := jobMetric.assignmentEnd.Sub(jobMetric.assignmentStart)
	crackTime := jobMetric.crackTime
	returnTime := jobMetric.returnEnd.Sub(jobMetric.returnStart)
	totalTime := dispatchTime + assignTime + crackTime + returnTime

	fmt.Println(resultTitle)
	fmt.Println(passwordTitle)
	fmt.Println("Dispatch:", dispatchTime.Seconds())
	fmt.Println("ChunkAssign:", assignTime.Seconds())
	fmt.Println("Crack:", crackTime.Seconds())
	fmt.Println("Return:", returnTime)
	fmt.Println("Total:", totalTime.Seconds())
	fmt.Println("=================================")
	fmt.Println()
}

func (c *Controller) printFinalResults(result string, err error) {
	resultTitle := "==== FINAL RESULTS (seconds) ===="
	var passwordTitle string
	if err != nil {
		passwordTitle = fmt.Sprintf("PASSWORD NOT FOUND: %s", err)
	} else {
		passwordTitle = fmt.Sprintf("PASSWORD: %s", result)
	}

	parseTime, crackTime, assignmentOverhead, dispatchOverhead, returnOverhead, checkpointOverhead := c.metric.EndToEndMetrics()
	totalTime := parseTime + crackTime + assignmentOverhead + dispatchOverhead + returnOverhead + checkpointOverhead

	rate, variance := c.metric.HeartbeatMetrics()

	fmt.Println(resultTitle)
	fmt.Println(passwordTitle)
	fmt.Println("Parse:", parseTime.Seconds())
	fmt.Println("Dispatch:", dispatchOverhead.Seconds())
	fmt.Println("ChunkAssign:", assignmentOverhead.Seconds())
	fmt.Println("Crack:", crackTime.Seconds())
	fmt.Println("Return:", returnOverhead.Seconds())
	fmt.Println("Checkpoint overhead:", checkpointOverhead.Seconds())
	fmt.Println("Total:", totalTime.Seconds())
	fmt.Println("Average Cracking Rate (attempts/s):", rate)
	fmt.Println("Heartbeat Variance:", variance)
	fmt.Println("=================================")
	fmt.Println()
}

func (c *Controller) displayJobResults(result string, err error, chunkID int, ts time.Time) {
	startPassword := shared.EncodeBase(c.chunks[chunkID].start, shared.SearchSpace)
	endPassword := shared.EncodeBase(c.chunks[chunkID].end, shared.SearchSpace)
	timings := c.chunkTimings[chunkID]

	var passwordString string
	chunkString := fmt.Sprintf("==== CHUNK: '%s' to '%s' RESULTS (seconds) ====", startPassword, endPassword)

	if err != nil {
		passwordString = fmt.Sprintf("PASSWORD NOT FOUND: %s", err)
	} else {
		passwordString = fmt.Sprintf("PASSWORD: %s", result)
	}

	c.prettyPrintResults(
		chunkString,
		passwordString,
		c.LatencyParse,
		timings.dispatchTime,
		timings.chunkAssignTime,
		timings.crackTime,
		timings.returnTime,
	)

	// report final results if password found
	if err != nil {
		return
	}

	finalString := "==== FINAL RESULTS (seconds) ===="
	var totaldispatch, totalChunkAssign, totalCrack, totalReturn time.Duration
	for _, timing := range c.chunkTimings {
		totaldispatch += timing.dispatchTime
		totalChunkAssign += timing.chunkAssignTime
		totalCrack += timing.crackTime
		totalReturn += timing.returnTime
	}
	c.prettyPrintResults(
		finalString,
		passwordString,
		c.LatencyParse,
		totaldispatch,
		totalChunkAssign,
		ts.Sub(*c.crackStart),
		totalReturn,
	)

	delta := 0
	for _, d := range c.deltaTimings {
		delta += d
	}
	averageDelta := float64(delta) / float64(len(c.deltaTimings))
	fmt.Printf("Average Delta (heartbeat/%ds): %f\n", c.Config.HeartbeatSeconds, averageDelta)
}

func (c *Controller) prettyPrintResults(
	title, password string,
	parse, dispatch, chunkAssign, crack, returnTime time.Duration,
) {
	total := parse + dispatch + chunkAssign + crack + returnTime

	fmt.Println(title)
	fmt.Println(password)
	fmt.Println("Parse:", parse.Seconds())
	fmt.Println("Dispatch:", dispatch.Seconds())
	fmt.Println("ChunkAssign:", chunkAssign.Seconds())
	fmt.Println("Crack:", crack.Seconds())
	fmt.Println("Return:", returnTime.Seconds())
	fmt.Println("Total:", total.Seconds())
	fmt.Println("=================================")
	fmt.Println()
}
