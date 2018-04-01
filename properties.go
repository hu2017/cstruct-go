package cstruct

import (
	"fmt"
	"reflect"
	"sync"
)

// StructProperties

type StructProperties struct {
	Prop []*Properties
}

var (
	propertiesMu  sync.RWMutex
	propertiesMap = make(map[reflect.Type]*StructProperties)
)

func GetProperties(t reflect.Type) *StructProperties {
	if t.Kind() != reflect.Struct {
		panic("cstruct: type must have kind struct")
	}

	propertiesMu.RLock()
	sprop, ok := propertiesMap[t]
	propertiesMu.RUnlock()
	if ok {
		return sprop
	}

	propertiesMu.Lock()
	sprop = getPropertiesLocked(t)
	propertiesMu.Unlock()
	return sprop
}

func getPropertiesLocked(t reflect.Type) *StructProperties {
	if prop, ok := propertiesMap[t]; ok {
		return prop
	}

	prop := new(StructProperties)
	propertiesMap[t] = prop
	prop.Prop = make([]*Properties, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		p := new(Properties)
		name := f.Name
		p.init(f.Type, name, "", &f)
		prop.Prop[i] = p
	}

	return prop
}

// Properties

type encoder func(p *Buffer, prop *Properties, base structPointer) error
type decoder func(p *Buffer, prop *Properties, base structPointer) error

type Properties struct {
	Name  string
	tag   int
	field field
	enc   encoder
	dec   decoder
	stype reflect.Type
	sprop *StructProperties
}

func (p *Properties) init(typ reflect.Type, name, tag string, f *reflect.StructField) {
	p.Name = name
	if f != nil {
		p.field = toField(f)
	}
	if typ.Kind() == reflect.Ptr {
		if t2 := typ.Elem(); t2.Kind() == reflect.Struct {
			p.stype = t2
			p.sprop = getPropertiesLocked(p.stype)
			p.enc = (*Buffer).enc_substruct_ptr
			p.dec = (*Buffer).dec_substruct_ptr
		}
		return
	}
	if typ.Kind() == reflect.Struct {
		p.stype = typ
		p.sprop = getPropertiesLocked(p.stype)
		p.enc = (*Buffer).enc_substruct
		p.dec = (*Buffer).dec_substruct
		return
	}
	if p.Parse(typ, f) == 0 {
		return
	}
	p.setEncAndDec(typ, f)

}

func (p *Properties) Parse(typ reflect.Type, f *reflect.StructField) int {
	switch typ.Kind() {
	case reflect.Bool:
		p.tag = 1
	case reflect.Int8, reflect.Uint8:
		p.tag = 2
	case reflect.Int16, reflect.Uint16:
		p.tag = 3
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		p.tag = 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		p.tag = 5
	case reflect.String:
		p.tag = 6
	case reflect.Slice:
		switch t2 := typ.Elem(); t2.Kind() {
		case reflect.Uint8: // []byte
			p.tag = 7
		}
	}
	if p.tag == 0 {
		panic("cstruct: unknow type. field name =" + f.Name)
	}
	return p.tag
}

func (p *Properties) setEncAndDec(typ reflect.Type, f *reflect.StructField) {
	p.enc = nil
	p.dec = nil

	switch p.tag {
	case 1:
		p.enc = (*Buffer).enc_bool
		p.dec = (*Buffer).dec_bool
	case 2:
		p.enc = (*Buffer).enc_uint8
		p.dec = (*Buffer).dec_uint8
	case 3:
		if CurrentByteOrder == LE {
			p.enc = (*Buffer).enc_uint16le
			p.dec = (*Buffer).dec_uint16le
		} else {
			p.enc = (*Buffer).enc_uint16be
			p.dec = (*Buffer).dec_uint16be
		}
	case 4:
		if CurrentByteOrder == LE {
			p.enc = (*Buffer).enc_uint32le
			p.dec = (*Buffer).dec_uint32le
		} else {
			p.enc = (*Buffer).enc_uint32be
			p.dec = (*Buffer).dec_uint32be
		}
	case 5:
		if CurrentByteOrder == LE {
			p.enc = (*Buffer).enc_uint64le
			p.dec = (*Buffer).dec_uint64le
		} else {
			p.enc = (*Buffer).enc_uint64be
			p.dec = (*Buffer).dec_uint64be
		}
	case 6:
		p.enc = (*Buffer).enc_string
		p.dec = (*Buffer).dec_string
	case 7:
		p.enc = (*Buffer).enc_binary
		p.dec = (*Buffer).dec_binary
	default:
		panic(fmt.Sprintf("unknow type! type = %d", p.tag))
	}
}
