package parse

import "strings"

type KeyValue struct {
	parent           *Section
	KeyName          string
	KeysValue        interface{}
	isCommented      bool
	Description      []string
	commentDelimiter string
	valueDelimiter   string
	nameDelimiter    string
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
			k.KeysValue = val
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
	return k
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
		// // define kv
		// header := &kv.Description[0]
		// // split kv and any comment
		// if keyVal, comment, found := strings.Cut(*header, ";"); found {
		// 	// println(comment)
		// 	// println(keyVal)
		// 	kv.Description[0] = strings.TrimSpace(comment)
		// 	header = &comment
		// 	if k, val, found := strings.Cut(keyVal, "="); found {
		// 		kv.KeysValue = val
		// 		k, c := strings.CutPrefix(k, "//")
		// 		k = strings.TrimSpace(k)
		// 		kv.isCommented = c
		// 		kv.KeyName = k
		// 	} else {
		// 		println("Unknown err: trying to split Key & Value")
		// 	}
		// } else {
		// 	println("Unknown err: trying to split KeyValue from trailing Comment")
		// }

		// kv.parent = parent
		// println(kv.keyName)
		Kvs = append(Kvs, kv)
	}
	return Kvs
}
