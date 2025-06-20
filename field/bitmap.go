package field

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cimartinez3/iso8583/utils"
)

const minBitmapLength = 8 // 64 bit, 8 bytes, or 16 hex digits
const maxBitmaps = 3

var _ Field = (*Bitmap)(nil)

type Bitmap struct {
	spec   *Spec
	bitmap *utils.Bitmap
	data   *Bitmap
}

func NewBitmap(spec *Spec) *Bitmap {
	return &Bitmap{
		spec:   spec,
		bitmap: utils.NewBitmap(64 * maxBitmaps),
	}
}

func (f *Bitmap) Spec() *Spec {
	return f.spec
}

func (f *Bitmap) SetSpec(spec *Spec) {
	f.spec = spec
}

func (f *Bitmap) SetBytes(b []byte) error {
	f.bitmap = utils.NewBitmapFromData(b)
	return nil
}

func (f *Bitmap) Bytes() ([]byte, error) {
	return f.bitmap.Bytes(), nil
}

func (f *Bitmap) String() (string, error) {
	return f.bitmap.String(), nil
}

func (f *Bitmap) Pack() ([]byte, error) {
	f.setBitmapFields()

	count := f.bitmapsCount()

	// here we have max possible bytes for the bitmap 8*maxBitmaps
	data, err := f.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve bytes: %v", err)
	}

	data = data[0 : 8*count]

	packed, err := f.spec.Enc.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode content: %v", err)
	}

	return packed, nil
}

// Unpack of the Bitmap field returns data of varied length
// if there is only primary bitmap (bit 1 is not set) we return only 8 bytes (or 16 for hex encoding)
// if secondary bitmap presents (bit 1 is set) we return 16 bytes (or 32 for hex encoding)
// and so on for maxBitmaps
func (f *Bitmap) Unpack(data []byte) (int, error) {
	minLen, err := f.spec.Pref.DecodeLength(minBitmapLength, data)
	if err != nil {
		return 0, fmt.Errorf("failed to decode length: %v", err)
	}

	rawBitmap := make([]byte, 0)
	read := 0

	// read max
	for i := 0; i < maxBitmaps; i++ {
		decoded, readDecoded, err := f.spec.Enc.Decode(data[read:], minLen)
		if err != nil {
			return 0, fmt.Errorf("failed to decode content for %d bitmap: %v", i+1, err)
		}
		read += readDecoded

		rawBitmap = append(rawBitmap, decoded...)
		bitmap := utils.NewBitmapFromData(decoded)

		// if no more bitmaps, exit loop
		if !bitmap.IsSet(1) {
			break
		}
	}

	f.bitmap = utils.NewBitmapFromData(rawBitmap)

	if f.data != nil {
		*(f.data) = *f
	}

	return read, nil
}

func (f *Bitmap) SetData(data interface{}) error {
	if data == nil {
		return nil
	}

	bmap, ok := data.(*Bitmap)
	if !ok {
		return fmt.Errorf("data does not match required *Bitmap type")
	}

	f.data = bmap
	if bmap.bitmap != nil {
		f.bitmap = bmap.bitmap
	}
	return nil
}

func (f *Bitmap) Reset() {
	f.bitmap = utils.NewBitmap(64 * maxBitmaps)
}

func (f *Bitmap) Set(i int) {
	f.bitmap.Set(i)
}

func (f *Bitmap) IsSet(i int) bool {
	return f.bitmap.IsSet(i)
}

func (f *Bitmap) Len() int {
	return f.bitmap.Len()
}

func (f *Bitmap) bitmapsCount() int {
	count := 1
	for i := 0; i < maxBitmaps; i++ {
		if f.IsSet(i*64 + 1) {
			count += 1
		}
	}

	return count
}

func (f *Bitmap) setBitmapFields() bool {
	// 2nd bitmap bits 65 -128
	// bitmap bit 1

	// 3rd bitmap bits 129-192
	// bitmap bit 65

	// start from the 2nd bitmap as for the 1st bitmap we don't need to set any bits
	for bitmapIndex := 2; bitmapIndex <= maxBitmaps; bitmapIndex++ {

		// are there fields for this (bitmapIndex) bitmap?
		bitmapStart := (bitmapIndex-1)*64 + 2 // we skip firt bit as it's for the next bitmap
		bitmapEnd := (bitmapIndex) * 64       //

		for i := bitmapStart; i <= bitmapEnd; i++ {
			bitmapBit := (bitmapIndex-2)*64 + 1
			if f.IsSet(i) {
				if bitmapIndex == 3 {
					f.Set(1)
				}
				f.Set(bitmapBit)
				break
			}
		}
	}

	return false
}

// Returns HEX encoded bitmap (if any)
func (f *Bitmap) MarshalJSON() ([]byte, error) {
	data, err := f.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve bytes: %v", err)
	}
	return json.Marshal(strings.ToUpper(hex.EncodeToString(data)))
}
