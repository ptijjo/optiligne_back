package ws

// Enqueue envoie dans une queue bornée ; false si pleine (drop).
func Enqueue(ch chan []byte, msg []byte) bool {
	select {
	case ch <- msg:
		return true
	default:
		return false
	}
}
