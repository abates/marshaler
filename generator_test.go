package main

import (
	"go/format"
	"strings"
	"testing"
)

type Test struct {
	name   string
	input  string
	output string
}

var tests = []Test{
	{"t1", t1_in, t1_out},
	{"t2", t2_in, t2_out},
	{"t3", t3_in, t3_out},
	{"t4", t4_in, t4_out},
	{"t5", t5_in, t5_out},
	{"t6", t6_in, t6_out},
	{"t7", t7_in, t7_out},
}

const t1_in = `
type T1 struct {
	length uint8
}
`
const t1_out = `
func (t *T1) UnmarshalBinary(data []byte) error {
	if len(data) < 1 {
		return io.EOF
	}
	t.length = uint8(data[0])
	return nil
}
`

const t2_in = `
type T2 struct {
	b1 uint8
	b2 uint8
}
`
const t2_out = `
func (t *T2) UnmarshalBinary(data []byte) error {
	if len(data) < 2 {
		return io.EOF
	}
	t.b1 = uint8(data[0])
	t.b2 = uint8(data[1])
	return nil
}
`

const t3_in = `
type T3 struct {
	b1 uint8
	b2 uint8
	b3 bool
}
`
const t3_out = `
func (t *T3) UnmarshalBinary(data []byte) error {
	if len(data) < 3 {
		return io.EOF
	}
	t.b1 = uint8(data[0])
	t.b2 = uint8(data[1])
	t.b3 = data[2] & 0x80 != 0
	return nil
}
`
const t4_in = `
type T4 struct {
	b1 uint8
	b2 uint8
	b3 bool
	b4 bool
}
`
const t4_out = `
func (t *T4) UnmarshalBinary(data []byte) error {
	if len(data) < 3 {
		return io.EOF
	}
	t.b1 = uint8(data[0])
	t.b2 = uint8(data[1])
	t.b3 = data[2] & 0x80 != 0
	t.b4 = data[2] & 0x40 != 0
	return nil
}
`

const t5_in = `
type T5 struct {
	b1 uint8
	b2 uint8
	b3 bool
	b4 bool
	b5 uint8
}
`
const t5_out = `
func (t *T5) UnmarshalBinary(data []byte) error {
	if len(data) < 3 {
		return io.EOF
	}
	t.b1 = uint8(data[0])
	t.b2 = uint8(data[1])
	t.b3 = data[2] & 0x80 != 0
	t.b4 = data[2] & 0x40 != 0
	t.b5 = uint8(data[2] & 0x3f)
	return nil
}
`

const t6_in = `
type T6 struct {
	b1 uint8
	b2 uint8
	b3 bool
	b4 bool
	b5 uint16
}
`
const t6_out = `
func (t *T6) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return io.EOF
	}
	t.b1 = uint8(data[0])
	t.b2 = uint8(data[1])
	t.b3 = data[2] & 0x80 != 0
	t.b4 = data[2] & 0x40 != 0
	t.b5 = binary.BigEndian.Uint16([]byte { data[2] & 0x3f, data[3] })
	return nil
}
`

const t7_in = `
type T7 struct {
	b1 uint8
	b2 uint8
	b3 bool
	b4 bool
	b5 uint64 ` + "`binary:length:48`" + `
}
`
const t7_out = `
func (t *T7) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return io.EOF
	}
	t.b1 = uint8(data[0])
	t.b2 = uint8(data[1])
	t.b3 = data[2] & 0x80 != 0
	t.b4 = data[2] & 0x40 != 0
	t.b5 = binary.BigEndian.Uint64([]byte { 0x00, 0x00, data[2] & 0x3f, data[3], data[4], data[5], data[6], data[7] })
	return nil
}
`

func TestGenerate(t *testing.T) {
	for _, test := range tests {
		var g Generator
		input := "package test\n" + test.input
		file := test.name + ".go"
		g.parseFile(file, input)

		tokens := strings.SplitN(test.input, " ", 3)
		if len(tokens) != 3 {
			t.Fatalf("%s: need type declaration on first line", test.name)
		}

		g.generate("BigEndian", tokens[1])
		output, _ := format.Source([]byte(test.output))
		need := string(output)
		got := string(g.format())

		if got != need {
			t.Errorf("%s: got\n====%s====\nexpected\n====%s====\n", test.name, got, need)
		}
	}
}
