package controller

// assignJob updates the worker information to point to the chunk.
//
// A new chunk will be created if not already available.
func (c *Controller) assignJob(workerID string, chunkID int) {
	c.chunksMutex.Lock()
	defer c.chunksMutex.Unlock()

	c.workers[workerID].ChunkID = chunkID

	if existingChunk, ok := c.chunks[chunkID]; !ok {
		start := uint64(chunkID) * uint64(c.Config.ChunkSize)
		end := start + uint64(c.Config.ChunkSize)

		c.chunks[chunkID] = &chunk{
			start: start, end: end,
			index:  start,
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
