package dsi

import (
	"fmt"
	"reflect"
)

type dsiOpenSession struct {
	Flags           uint8
	Command         uint8
	RequestId       uint16
	ErrorCode       uint32
	TotalDataLength uint32
	Reserved        uint32
	Options         []option
}

type option struct {
	Type   uint8
	Length uint8
	Data   []byte
}

func marshal(val interface{}) []byte {
	var out []byte
	v := reflect.ValueOf(val)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		t := field.Type()
		switch t.Kind() {
		case reflect.Uint32:
			u32 := uint32(field.Uint())
			out = append(out, byte(u32>>24))
			out = append(out, byte(u32>>16))
			out = append(out, byte(u32>>8))
			out = append(out, byte(u32))
		case reflect.Uint16:
			u16 := uint16(field.Uint())
			out = append(out, byte(u16>>8))
			out = append(out, byte(u16))
		case reflect.Uint8:
			u8 := uint8(field.Uint())
			out = append(out, u8)
		case reflect.Slice:
			switch t.Elem().Kind() {
			case reflect.Uint8:
				length := field.Len()
				for j := 0; j < length; j++ {
					out = append(out, byte(field.Index(j).Uint()))
				}
			case reflect.Struct:
				for j := 0; j < field.Len(); j++ {
					out = append(out, marshal(field.Index(j))...)
				}
			default:
				panic("slice of unknown type " + t.Elem().Kind().String())
			}

		default:
			panic("unknown type " + t.Kind().String())
		}
	}
	return out
}

func unmarshal(out interface{}, data []byte) error {
        v := reflect.ValueOf(out).Elem()
        for i := 0; i < v.NumField(); i++ {
                field := v.Field(i)
                t := field.Type()
                switch t.Kind() {
		case reflect.Uint8:
			field.SetUint(uint64(data[0]))
			data = data[1:]
		case reflect.Uint16:
			field.SetUint(uint64(uint16(data[0] << 8) + uint16(data[1])))
			data = data[2:]
		case reflect.Uint32:
			field.SetUint(uint64(uint32(data[0] << 24) + uint32(data[1] << 16) + uint32(data[2] << 8) + uint32(data[3])))
			data = data[4:]
		case reflect.Slice:
			switch t.Elem().Kind() {
			default:
				return fmt.Errorf("unsupported slice of type: %v", t.Elem().Kind())
			}
		default:
			return fmt.Errorf("unsupported field: %v", t)
		}
	}
	return nil
}

func decode(data []byte) (interface{}, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("data is too short: %v", data)
	}
	var msg interface{}
	switch data[1] {
//        case DSICloseSession:
        //case DSICommand:
        //DSIGetStatus    = 3
        case DSIOpenSession:
		msg = new(dsiOpenSession)
        //DSITickle       = 5
        //DSIWrite        = 6
        //DSIAttention    = 8
	default:
		return nil, fmt.Errorf("unknown packet: %v", data)
	}
	if err := unmarshal(msg, data); err != nil {
		return nil, err
	}
	return msg, nil
}	
