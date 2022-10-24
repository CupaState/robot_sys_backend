package interfaces

type Server interface {
	Start()
	InterruptGracefulShutdown()
}
