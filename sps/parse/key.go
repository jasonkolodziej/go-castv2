package parse

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

type KeyValues []KeyValue
type KeyValue struct {
	parent           *Section
	KeyName          string
	KeysValue        any // interface{}
	isCommented      bool
	Description      []string
	commentDelimiter string
	valueDelimiter   string
	nameDelimiter    string
	valueType        reflect.Kind
}

func (k KeyValue) KvIsCommented() bool { return k.isCommented }

func KvIsCommented(k KeyValue) bool { return k.isCommented }

func CreateKvLines(rawKv, sep string) []string {
	if sep == "" {
		sep = "\n"
	}
	return NoEmpty(strings.Split(rawKv, sep))
}

func (k *KeyValue) SetDelimiters(nameDelimiter, valueDelimiter, commentDelimiter string) {
	k.nameDelimiter = nameDelimiter
	k.commentDelimiter = commentDelimiter
	k.valueDelimiter = valueDelimiter
}

func (k *KeyValue) GetDelimitersForAssertion() []string {
	return []string{k.nameDelimiter, k.valueDelimiter}
}

func (k *KeyValue) CreateKeyValue() *KeyValue {
	header := k.Description[0]
	// split kv and any comment
	if keyVal, comment, found := strings.Cut(header, k.valueDelimiter); found {
		// println(comment)
		// println(keyVal)
		k.Description[0] = strings.TrimSpace(comment)
		header = comment
		if key, val, found := strings.Cut(keyVal, k.nameDelimiter); found {
			k.KeysValue = strings.TrimSpace(val)
			key, c := strings.CutPrefix(key, k.commentDelimiter)
			key = strings.TrimSpace(key)
			k.isCommented = c
			k.KeyName = key
		} else {
			println("Unknown err: trying to split Key & Value")
		}
	} else {
		println("Unknown err: trying to split KeyValue from trailing Comment")
	}
	k.determineType()
	return k
}

func (k *KeyValue) determineType() {
	vString := k.KeysValue.(string)
	// val := reflect.ValueOf(k.KeysValue)
	if strings.Index(vString, "\"") == 0 && // * is the first & last char of the string a quote
		strings.LastIndex(vString, "\"") == len(vString)-1 { // * mark as string
		k.valueType = reflect.String
		return
	}
	if _, err := strconv.ParseUint(vString, 10, 0); err == nil {
		k.valueType = reflect.Uint
	} else if _, err := strconv.ParseInt(vString, 10, 0); err == nil {
		k.valueType = reflect.Int
	} else if _, err := strconv.ParseFloat(vString, 32); err == nil {
		k.valueType = reflect.Float32
	} else if _, err := strconv.ParseFloat(vString, 64); err == nil {
		k.valueType = reflect.Float64
	} else {
		k.valueType = reflect.UnsafePointer
	}
}

func (k *KeyValue) SetValue(val any) error {
	t := reflect.TypeOf(val).Kind()
	if t != k.valueType {
		return fmt.Errorf("setting value TypeOf.Kind: %s, expected kind %s", t, k.valueType)
	}

	k.KeysValue = val
	return nil
}

func CreateKvs(allLines []string, kvIdx []int, parent *Section) []KeyValue {
	l := len(kvIdx)
	var Kvs []KeyValue
	for i, v := range kvIdx {
		var kv KeyValue
		if p := i + 1; p <= (l - 1) {
			// next index
			p = kvIdx[i+1]
			// lines from current index to next
			kv = KeyValue{
				parent:      parent,
				Description: allLines[v:p],
			}
		} else {
			p = kvIdx[i]
			// lines from last index
			kv = KeyValue{
				parent:      parent,
				Description: allLines[p:],
			}
		}
		kv.SetDelimiters("=", ";", "//")
		kv.CreateKeyValue()
		Kvs = append(Kvs, kv)
	}
	return Kvs
}

func (kv *KeyValue) Type() reflect.Kind {
	return kv.valueType
}

func (kv *KeyValue) SetCommented() {
	kv.isCommented = true
}
func (kv *KeyValue) SetUncommented() {
	kv.isCommented = false
}

func (k *KeyValue) WriteTo(w io.Writer) (int64, error) {
	l := len(k.Description)
	descs := Append(k.Description, "// ")
	var full string
	// var inLineComment string
	var kvLine string
	if k.KvIsCommented() {
		kvLine = "//\t" + k.KeyName + " = " + k.KeysValue.(string) + "; "
	} else {
		kvLine = "\t" + k.KeyName + " = " + k.KeysValue.(string) + "; "
	}
	if l > 1 {
		kvLine += descs[0] + "\n"
		full = kvLine + strings.Join(descs[1:], "\n") + "\n"
	} else if l == 1 {
		kvLine += descs[0] + "\n"
	}
	if full == "" {
		full = kvLine
	}
	n, err := w.Write([]byte(full))
	return int64(n), err
}
