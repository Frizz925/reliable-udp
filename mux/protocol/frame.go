package protocol

type Frame interface {
	Bytes() []byte
}
