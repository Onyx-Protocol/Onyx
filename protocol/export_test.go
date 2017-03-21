package protocol

func SetChainHeight(c *Chain, height uint64) {
	c.setHeight(height)
}

func GetChainHeight(c *Chain) uint64 {
	return c.state.height
}
