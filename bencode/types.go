package bencode

type Value interface{}

type Integer int64
type String string
type List []Value
type Dict map[string]Value
