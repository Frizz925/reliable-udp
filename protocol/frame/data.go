package frame

type Serializer interface {
	Bytes() []byte
}

type Data interface {
	Serializer
	Type() FrameType
}
