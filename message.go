package iso8583

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/cimartinez3/iso8583/field"
)

const (
	bitmapLength = 64
)

type Message struct {
	spec      *MessageSpec
	data      interface{}
	dataValue *reflect.Value
	bitmap    *field.Bitmap

	// stores all fields according to the spec
	fields map[int]field.Field
	// tracks which fields were set
	fieldsMap map[int]struct{}
}

func NewMessage(spec *MessageSpec) *Message {
	fields := spec.CreateMessageFields()

	return &Message{
		fields:    fields,
		spec:      spec,
		fieldsMap: map[int]struct{}{},
	}
}

func (m *Message) Data() interface{} {
	return m.data
}

func (m *Message) SetData(data interface{}) error {
	m.data = data

	if m.data == nil {
		return nil
	}

	dataStruct := reflect.ValueOf(data)
	if dataStruct.Kind() == reflect.Ptr || dataStruct.Kind() == reflect.Interface {
		// get the struct
		dataStruct = dataStruct.Elem()
	}

	if reflect.TypeOf(dataStruct).Kind() != reflect.Struct {
		return fmt.Errorf("failed to set data as struct is expected, got: %s", reflect.TypeOf(dataStruct).Kind())
	}

	m.dataValue = &dataStruct
	return nil
}

func (m *Message) Bitmap() *field.Bitmap {
	if m.bitmap != nil {
		return m.bitmap
	}

	m.bitmap = m.fields[1].(*field.Bitmap)
	m.bitmap.Reset()
	m.fieldsMap[1] = struct{}{}

	return m.bitmap
}

func (m *Message) MTI(val string) {
	m.fieldsMap[0] = struct{}{}
	m.fields[0].SetBytes([]byte(val))
}

func (m *Message) GetSpec() *MessageSpec {
	return m.spec
}

func (m *Message) Field(id int, val string) error {
	if f, ok := m.fields[id]; ok {
		m.fieldsMap[id] = struct{}{}
		return f.SetBytes([]byte(val))
	}
	return fmt.Errorf("failed to set field %d. ID does not exist", id)
}

func (m *Message) BinaryField(id int, val []byte) error {
	if f, ok := m.fields[id]; ok {
		m.fieldsMap[id] = struct{}{}
		return f.SetBytes(val)
	}
	return fmt.Errorf("failed to set binary field %d. ID does not exist", id)
}

func (m *Message) GetMTI() (string, error) {
	// check index
	return m.fields[0].String()
}

func (m *Message) GetString(id int) (string, error) {
	if f, ok := m.fields[id]; ok {
		m.fieldsMap[id] = struct{}{}
		return f.String()
	}
	return "", fmt.Errorf("failed to get string for field %d. ID does not exist", id)
}

func (m *Message) GetBytes(id int) ([]byte, error) {
	if f, ok := m.fields[id]; ok {
		m.fieldsMap[id] = struct{}{}
		return f.Bytes()
	}
	return nil, fmt.Errorf("failed to get bytes for field %d. ID does not exist", id)
}

func (m *Message) GetField(id int) field.Field {
	return m.fields[id]
}

// Fields returns the map of the set fields
func (m *Message) GetFields() map[int]field.Field {
	fields := map[int]field.Field{}
	for i := range m.fieldsMap {
		fields[i] = m.GetField(i)
	}
	return fields
}

func (m *Message) Pack() ([]byte, error) {
	packed := []byte{}
	m.Bitmap().Reset()

	ids, err := m.setPackableDataFields()
	if err != nil {
		return nil, fmt.Errorf("failed to pack message: %w", err)
	}

	for _, id := range ids {
		// indexes 0 and 1 are for mti and bitmap
		// regular field number startd from index 2
		if id < 2 {
			continue
		}
		m.Bitmap().Set(id)
	}

	// pack fields
	for _, i := range ids {
		field, ok := m.fields[i]
		if !ok {
			return nil, fmt.Errorf("failed to pack field %d: no specification found", i)
		}
		packedField, err := field.Pack()
		if err != nil {
			return nil, fmt.Errorf("failed to pack field %d (%s): %v", i, field.Spec().Description, err)
		}
		packed = append(packed, packedField...)
	}

	return packed, nil
}

func (m *Message) Unpack(src []byte) error {
	var off int

	m.fieldsMap = map[int]struct{}{}
	m.Bitmap().Reset()

	// unpack MTI
	if m.dataValue != nil {
		if err := m.setUnpackableDataField(0); err != nil {
			return err
		}
	}
	read, err := m.fields[0].Unpack(src)
	if err != nil {
		return fmt.Errorf("failed to unpack MTI: %v", err)
	}

	off = read

	// unpack Bitmap
	read, err = m.fields[1].Unpack(src[off:])
	if err != nil {
		return fmt.Errorf("failed to unpack bitmapt: %v", err)
	}

	off += read

	for i := 2; i <= m.Bitmap().Len(); i++ {
		if m.Bitmap().IsSet(i) && !IsBitmapFlag(i) {
			fl, ok := m.fields[i]
			if !ok {
				return fmt.Errorf("failed to unpack field %d: no specification found", i)
			}

			if m.dataValue != nil {
				if err := m.setUnpackableDataField(i); err != nil {
					return err
				}
			}

			m.fieldsMap[i] = struct{}{}
			read, err = fl.Unpack(src[off:])
			if err != nil {
				return fmt.Errorf("failed to unpack field %d (%s): %v", i, fl.Spec().Description, err)
			}

			off += read
		}
	}

	return nil
}

func (m *Message) MarshalJSON() ([]byte, error) {
	// by packing the message we will generate bitmap
	// create HEX representation
	// and validate message against the spec
	if _, err := m.Pack(); err != nil {
		return nil, err
	}

	// get only filds that were set
	jsonData := field.OrderedMap(m.GetFields())

	return json.Marshal(jsonData)
}

func (m *Message) setPackableDataFields() ([]int, error) {
	// Index 0 and 1 represent the MTI and bitmap respectively.
	// These fields are assumed to be always populated.
	populatedFieldIDs := []int{0, 1}

	for id, field := range m.fields {
		// represents the bitmap
		if id == 1 {
			continue
		}

		// These fields are set using the typed API
		if m.dataValue != nil {
			dataField := m.dataFieldValue(id)
			// no non-nil data field was found with this name
			if dataField == (reflect.Value{}) || dataField.IsNil() {
				continue
			}
			if err := field.SetData(dataField.Interface()); err != nil {
				return nil, fmt.Errorf("failed to set data for field %d: %w", id, err)
			}

			// mark field as set
			m.fieldsMap[id] = struct{}{}
		}

		// These fields are set using the untyped API
		_, ok := m.fieldsMap[id]
		// We don't wish set the MTI again, hence we ignore the 0
		// index
		if (ok || m.dataValue != nil) && id != 0 {
			populatedFieldIDs = append(populatedFieldIDs, id)
		}
	}
	sort.Ints(populatedFieldIDs)

	return populatedFieldIDs, nil
}

func (m *Message) setUnpackableDataField(id int) error {
	specField, ok := m.fields[id]
	if !ok {
		return fmt.Errorf("failed to unpack field %d: no specification found", id)
	}

	dataField := m.dataFieldValue(id)
	// no data field was found with this name
	if dataField == (reflect.Value{}) {
		return nil
	}

	isNil := dataField.IsNil()
	if isNil {
		dataField.Set(reflect.New(dataField.Type().Elem()))
	}
	if err := specField.SetData(dataField.Interface()); err != nil {
		return fmt.Errorf("failed to set data for field %d: %w", id, err)
	}

	return nil
}

func (m *Message) dataFieldValue(id int) reflect.Value {
	return m.dataValue.FieldByName(fmt.Sprintf("F%d", id))
}

// IsBitmapFlag validate if field is a bitmap flag
// The presence of the second bitmap is defined in the first bit of the first bitmap (bit 1).
// The presence of the third bitmap is defined in the first bit of the second bitmap (bit 65).
// The presence of the fourth bitmap is defined in the first bit of the third bitmap (bit 129).
func IsBitmapFlag(i int) bool {
	return (i % bitmapLength) == 1
}
