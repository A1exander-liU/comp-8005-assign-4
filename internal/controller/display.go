package controller

import (
	"fmt"

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
	fmt.Println("Return:", returnTime.Seconds())
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
