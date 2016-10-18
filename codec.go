package panda

import (
	///TODO: when the author add support for encoding use this package instead of encoding/json: "github.com/a8m/djson"
	"encoding/json"
	"fmt"
	"github.com/fatih/structs"
	"github.com/kataras/go-errors"
	"github.com/ugorji/go/codec"
	"math/rand"
	"reflect"
	"strconv"
	"time"
)

// Codec ...TODO:
type Codec interface {
	// Serialize TOOD:
	Serialize(interface{}) ([]byte, error)
	// Deserialize TOOD:
	Deserialize([]byte, interface{}) error
}

type jsonCodec struct{}

var (
	// DefaultCodec the default codec
	DefaultCodec       = &jsonCodec{}
	_            Codec = &jsonCodec{}
)

// Serialize TOOD:
func (j *jsonCodec) Serialize(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Deserialize TOOD: PASS THE POINTER TO THE OUT MANUALLY ([]byte, &Request{})
func (j *jsonCodec) Deserialize(in []byte, out interface{}) error {
	return json.Unmarshal(in, out) // faster when we have the []byte already
}

//  codecJsonCodec codec library supposed to be faster than standard json, we need to test it before use it.
type codecJSONCodec struct{}

var (
	_                Codec = &codecJSONCodec{}
	codecJSONHandler       = &codec.JsonHandle{}
)

func (j *codecJSONCodec) Serialize(v interface{}) (out []byte, err error) {
	dec := codec.NewEncoderBytes(&out, codecJSONHandler) // we can't use writer*
	err = dec.Encode(v)
	return
}

// Deserialize TOOD: PASS THE POINTER TO THE OUT MANUALLY ([]byte, &Request{})
func (j *codecJSONCodec) Deserialize(in []byte, out interface{}) error {
	dec := codec.NewDecoderBytes(in, codecJSONHandler) // we can't use writer*
	return dec.Decode(out)
}

var (
	errWrongKindValue = errors.New("Wrong kind of value %s")
	errWrongKindFrom  = errors.New("Expected a map[string]interface{} or map[interface{}]interface{}, instead got %#v")
)

func decodeMap(v interface{}, m map[string]interface{}) error {
	s := structs.New(v)
	for k, v := range m {
		if f, ok := s.FieldOk(k); ok {
			valKind := f.Value()
			if valKind == reflect.Int {
				f.Set(int(MustDecodeInt(f.Value())))
			} else if valKind == reflect.Uint64 {
				f.Set(MustDecodeInt(f.Value()))
			} else if valKind == reflect.Float64 {
				f.Set(float64(MustDecodeInt(f.Value())))
			} else {
				f.Set(v)
			}

		} else {
			return errWrongKindValue.Format(v)
		}
	}
	return nil
}

// DecodeResult converts a map to struct, TOOD: make it like panda.Result to be more explicit which wil lbe returned from handlers too.?
// but from builded handlers no the handler user gives to make it easier to write a  handler
func DecodeResult(v interface{}, from interface{}) error {
	if m, isMap := from.(map[string]interface{}); isMap {
		return decodeMap(v, m)
	} else if m, isMap := from.(map[interface{}]interface{}); isMap {
		mapIntStr := make(map[string]interface{}, len(m))
		for key, value := range m {
			switch key := key.(type) {
			case string:
				switch value := value.(type) {
				case string:
					mapIntStr[key] = value
				}
			}
		}
		return decodeMap(v, mapIntStr)
	}
	return errWrongKindFrom.Format(from)
}

// Expect receives a struct and the 'client' .Do call result and set that result which is
func Expect(v interface{}, do func(string, ...Arg) (interface{}, error)) func(string, ...Arg) interface{} {
	return func(statement string, args ...Arg) interface{} {
		responseResult, _ := do(statement, args...)
		// check if jsonObject is already a pointer, if yes then pass as it's
		_, isMap := responseResult.(map[string]interface{})
		_, isMapII := responseResult.(map[interface{}]interface{})
		if !isMap && !isMapII {
			return responseResult // it's a standard type (int,float,uint,string,bool...)
		}

		if reflect.TypeOf(v).Kind() == reflect.Ptr {
			DecodeResult(v, responseResult)
		} else {
			return nil
		}

		// we don't keep the same return structure as the .Do , Except should called if no error expected and also to be easier to cast it in one line
		return v

	}

}

// MustDecodeInt receives an unknown type of and it should return its uint64 representation, returns 0 on error
func MustDecodeInt(v interface{}) int {
	n, err := DecodeInt(v)
	if err != nil {
		//panic(err)
		return 0
	}
	return n
}

// DecodeInt receives any type and tries to return the int
func DecodeInt(v interface{}) (n int, err error) {
	if i, isInt := v.(int); isInt {
		n = i
	} else if f64, isF64 := v.(float64); isF64 {
		n = int(f64)
	} else if ui64, isUI64 := v.(uint64); isUI64 {
		n = int(ui64)
	} else if s, ok := v.(string); ok {
		n, err = strconv.Atoi(s)
	} else if b, ok := v.(bool); ok {
		if b {
			n = 1
		}
	} else if by, ok := v.([]byte); ok {
		n, err = strconv.Atoi(string(by))
	} else {
		err = fmt.Errorf("%#v is not supported type as Connection's ID(sent as uint64), check your decoder/encoder documantation", v)
	}
	return
}

//
const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// Random takes a parameter (int) and returns random slice of byte
// ex: var randomstrbytes []byte; randomstrbytes = utils.Random(32)
func Random(n int) []byte {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return b
}

// RandomString accepts a number(10 for example) and returns a random string using simple but fairly safe random algorithm
func RandomString(n int) string {
	return string(Random(n))
}
