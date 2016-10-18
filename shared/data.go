package shared

import (
	"bytes"
	"encoding/json"
	"github.com/valyala/bytebufferpool"
	"io"
	"strings"
)

// Payload TODO:
type Payload struct {
	OS     string
	GOPATH string
}

// Question TODO:
type Question struct {
	Text    string
	Payload Payload
	Answer  chan Answer `json:"-"`
}

// Answer TODO:
type Answer struct {
	Text string
}

// ofc I'll code all these to be faster, we can't just init a new encoder on each message :)
// also the user will be able to set custom codecs, I will write all these on a dream folder better
var buffer bytebufferpool.Pool

// Serialize TOOD:
func (q Question) Serialize() ([]byte, error) {
	w := buffer.Get()
	err := json.NewEncoder(w).Encode(q)
	result := w.Bytes()
	buffer.Put(w)
	return result, err
}

// SerializeTo TOOD:
func (q Question) SerializeTo(w io.Writer) error {
	return json.NewEncoder(w).Encode(q)
}

// DeserializeQuestion TODO:
func DeserializeQuestion(b []byte) (Question, error) {
	w := new(bytes.Buffer)
	w.Write(b)
	q := Question{}
	err := json.NewDecoder(w).Decode(&q)
	return q, err
}

// DeserializeQuestionFrom TODO:
func DeserializeQuestionFrom(r io.Reader) (Question, error) {
	q := Question{}
	err := json.NewDecoder(r).Decode(&q)
	return q, err
}

// Serialize TODO:
func (ans Answer) Serialize() ([]byte, error) {
	w := buffer.Get()
	err := json.NewEncoder(w).Encode(ans)
	result := w.Bytes()
	buffer.Put(w)
	return result, err
}

// SerializeTo TODO:
func (ans Answer) SerializeTo(w io.Writer) error {
	return json.NewEncoder(w).Encode(ans)
}

// DeserializeAnswer TODO:
func DeserializeAnswer(b []byte) (Answer, error) {
	w := new(bytes.Buffer)
	w.Write(b)
	ans := Answer{}
	err := json.NewDecoder(w).Decode(&ans)
	return ans, err
}

// DeserializeAnswerFrom TODO:
func DeserializeAnswerFrom(r io.Reader) (Answer, error) {
	ans := Answer{}
	err := json.NewDecoder(r).Decode(&ans)
	return ans, err
}

// ParseQuestionText TODO:
func ParseQuestionText(text string) string {
	qUTF8 := strings.Map(func(r rune) rune {
		if r >= 32 && r < 127 {
			return r
		}
		return -1
	}, text)
	qUTF8 = strings.TrimSpace(qUTF8)

	return qUTF8
}
