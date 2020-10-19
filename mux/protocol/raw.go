package protocol

type Raw []byte

func (r Raw) Bytes() []byte {
	return []byte(r)
}
