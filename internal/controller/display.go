package controller

import (
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

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
