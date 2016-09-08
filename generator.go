package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"math"
	"strconv"
	"strings"
)

type Field struct {
	g          *Generator
	name       string
	fieldType  string
	offset     int
	typeLength int
	length     int
}

var bitmasks = []byte{
	0x80,
	0x40,
	0x20,
	0x10,
	0x08,
	0x04,
	0x02,
	0x01,
}

var masks = []byte{
	0xff,
	0x7f,
	0x3f,
	0x1f,
	0x0f,
	0x07,
	0x03,
	0x01,
}

func (f *Field) unmarshal() string {
	offsetByte := f.offset / 8
	offsetBit := f.offset % 8
	b := []string{}
	for i := 0; i < f.length; i += 8 {
		if i == 0 && f.length > 1 && offsetBit > 0 {
			b = append(b, fmt.Sprintf("data[%d] & 0x%02x", offsetByte+i, masks[offsetBit]))
		} else {
			b = append(b, fmt.Sprintf("data[%d]", offsetByte+(i/8)))
		}
	}

	if f.length < f.typeLength {
		for i := 0; i < (f.typeLength - f.length); i += 8 {
			b = append([]string{"0x00"}, b...)
		}
	}

	switch f.fieldType {
	case "uint8":
		return fmt.Sprintf("%s(%s)", f.fieldType, b[0])
	case "uint16":
		return fmt.Sprintf("binary.%s.Uint16([]byte{%s})", f.g.byteOrder, strings.Join(b[0:2], ","))
	case "uint32":
		return fmt.Sprintf("binary.%s.Uint32([]byte{%s})", f.g.byteOrder, strings.Join(b[0:4], ","))
	case "uint64":
		return fmt.Sprintf("binary.%s.Uint64([]byte{%s})", f.g.byteOrder, strings.Join(b[0:8], ","))
	case "bool":
		return fmt.Sprintf("%s & 0x%02x != 0", b[0], bitmasks[offsetBit])
	default:
		log.Fatalf("Don't know how to unmarshal type %s", f.name)
	}
	return ""
}

func (f *Field) parse(field *ast.Field) {
	f.name = field.Names[0].Name
	f.fieldType = field.Type.(*ast.Ident).Name
	switch f.fieldType {
	case "uint8":
		f.typeLength = 8
	case "uint16":
		f.typeLength = 16
	case "uint32":
		f.typeLength = 32
	case "uint64":
		f.typeLength = 64
	case "bool":
		f.typeLength = 1
	default:
		log.Fatalf("Don't know how to parse type %s", f.fieldType)
	}

	f.length = f.typeLength
	if field.Tag != nil {
		tag := strings.Trim(field.Tag.Value, "`")
		if strings.HasPrefix(tag, "binary:") {
			tokens := strings.Split(tag, ":")
			switch tokens[1] {
			case "length":
				l, err := strconv.ParseInt(tokens[2], 10, 32)
				if err != nil {
					log.Fatalf("%v", err)
				}
				f.length = int(l)
			default:
				log.Fatalf("Don't know tag %s", tokens[1])
			}
		}
	}
}

type Generator struct {
	buf       bytes.Buffer
	byteOrder string
	files     []*ast.File
	pkg       string
	minLength int
}

func (g *Generator) format() []byte {
	src, _ := format.Source(g.buf.Bytes())
	return src
}

func (g *Generator) Printf(format string, args ...interface{}) {
	//fmt.Printf(format, args...)
	fmt.Fprintf(&g.buf, format, args...)
}

func (g *Generator) process(typeName string, str *ast.StructType) {
	var fields []*Field
	length := 0
	for _, field := range str.Fields.List {
		f := new(Field)
		f.g = g
		f.parse(field)
		f.offset = length
		fields = append(fields, f)
		if length%8 > 0 && f.length > 1 {
			length += (f.length - (length % 8))
		} else {
			length += f.length
		}
	}
	minLength := int(math.Ceil(float64(length) / 8))
	varname := strings.ToLower(typeName[0:1])
	g.Printf("\nfunc (%s *%s) UnmarshalBinary(data []byte) error {\n", varname, typeName)
	g.Printf("if len(data) < %d {\nreturn io.EOF\n}\n", minLength)
	for _, field := range fields {
		g.Printf("%s.%s = %s\n", varname, field.name, field.unmarshal())
	}
	g.Printf("return nil\n}\n")
}

func (g *Generator) generate(byteOrder, typeName string) {
	g.byteOrder = byteOrder
	for _, file := range g.files {
		ast.Inspect(file, func(node ast.Node) bool {
			if tnode, ok := node.(*ast.TypeSpec); ok && tnode.Name.Name == typeName {
				if snode, ok := tnode.Type.(*ast.StructType); ok {
					g.process(typeName, snode)
					return false
				}
			}
			return true
		})
	}
}

func (g *Generator) parseFile(name string, text interface{}) {
	fs := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fs, name, text, 0)
	if err != nil {
		log.Fatalf("parsing package: %s: %s", name, err)
	}

	g.files = append(g.files, parsedFile)
	g.pkg = parsedFile.Name.Name
}
