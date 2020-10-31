package signalr

import (
	"fmt"
)

const recordSeparatorCode = 0x1e

var mFormat MessageFormat

func init() {
	mFormat = MessageFormat{}
}

// MessageFormat ...
type MessageFormat struct{}

func (f *MessageFormat) write(m string) string {
	return fmt.Sprintf("%s%s", m, string(recordSeparatorCode))
}

func (f *MessageFormat) parse(m []byte) []byte {
	return m[:len(m)-1]
}
