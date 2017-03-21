package protocol

func SetChainHeight(c *Chain, height uint64) {
	c.setHeight(height)
}
