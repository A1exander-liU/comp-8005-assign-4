package controller

// getJob returns a job id that is available to be worked on.
//
// It will priortise existing jobs that are currently unassigned
// before attempting to create a new job.
func (c *Controller) getJob() int {
	for jobID, chunk := range c.chunks {
		if chunk.status == ChunkUnassigned {
			return jobID
		}
	}

	var nextJobID int
	c.nextChunkIDMutex.Lock()
	nextJobID = c.nextChunkID
	c.nextChunkID += 1
	c.nextChunkIDMutex.Unlock()

	return nextJobID
}

// assignJob updates the worker information to point to the chunk.
//
// A new chunk will be created if not already available.
func (c *Controller) assignJob(workerID string, chunkID int) {
	c.workers[workerID].ChunkID = chunkID

	if existingChunk, ok := c.chunks[chunkID]; !ok {
		start := uint64(chunkID) * uint64(c.Config.ChunkSize)
		end := start + uint64(c.Config.ChunkSize)

		c.chunks[chunkID] = &chunk{
			start: start, end: end,
			status: ChunkAssigned,
		}
	} else {
		existingChunk.status = ChunkAssigned
	}
}

// revokeJob updates the worker information to not point to the chunk.
//
// The chunk was also be changed back to the unassigned state.
func (c *Controller) revokeJob(workerID string, chunkID int) {
	c.workers[workerID].ChunkID = -1

	if chunk, ok := c.chunks[chunkID]; ok {
		chunk.status = ChunkUnassigned
	}
}

// reallocateJob assigns to job of oldWorkerID to newWorkerID.
//
// This will also delete the old worker well.
func (c *Controller) reallocateJob(oldWorkerID, newWorkerID string) {
	c.workers[newWorkerID] = c.workers[oldWorkerID]
	delete(c.workers, oldWorkerID)
}
