package codec

type ICodec interface {
	// EncodeStruct has no error return: implementations swallow marshal
	// errors (return nil/empty on failure).
	EncodeStruct(data interface{}) []byte
	DecodeStruct(data []byte, target interface{}) error
}
