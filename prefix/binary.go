package prefix

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/franizus/iso8583/encoding"
)

type binaryVarPrefixer struct {
	Digits int
}

var Binary = Prefixers{
	Fixed: &binaryFixedPrefixer{},
	L:     &binaryVarPrefixer{1},
	LL:    &binaryVarPrefixer{2},
	LLL:   &binaryVarPrefixer{3},
	LLLL:  &binaryVarPrefixer{4},
}

type binaryFixedPrefixer struct {
}

func (p *binaryFixedPrefixer) EncodeLength(fixLen, dataLen int) ([]byte, error) {
	if dataLen != fixLen {
		return nil, fmt.Errorf("field length: %d should be fixed: %d", dataLen, fixLen)
	}

	return []byte{}, nil
}

func (p *binaryFixedPrefixer) DecodeLength(fixLen int, data []byte) (int, error) {
	return fixLen, nil
}

func (p *binaryFixedPrefixer) Length() int {
	return 0
}

func (p *binaryFixedPrefixer) Inspect() string {
	return "Binary.Fixed"
}

func (p *binaryVarPrefixer) EncodeLength(maxLen, dataLen int) ([]byte, error) {
	if dataLen > maxLen {
		return nil, fmt.Errorf("field length: %d is larger than maximum: %d", dataLen, maxLen)
	}

	if len(strconv.Itoa(dataLen)) > p.Digits {
		return nil, fmt.Errorf("number of digits in length: %d exceeds: %d", dataLen, p.Digits)
	}

	res, err := encoding.Binary.Encode([]byte{byte(dataLen)})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *binaryVarPrefixer) DecodeLength(maxLen int, data []byte) (int, error) {
	if len(data) < p.Length() {
		return 0, fmt.Errorf("length mismatch: want to read %d bytes, get only %d", p.Length(), len(data))
	}

	bDigits, _, err := encoding.Binary.Decode(data[:p.Length()], p.Length())
	if err != nil {
		return 0, err
	}

	// Just when the field length is LLLL
	var dataLen int

	if p.Digits == 4 {
		parsedInt, err := strconv.ParseInt(hex.EncodeToString(bDigits), 16, 64)
		if err != nil {
			return 0, err
		}

		dataLen = int(parsedInt)
	} else {
		dataLen = int(bDigits[0])
	}

	if dataLen > maxLen {
		return 0, fmt.Errorf("data length %d is larger than maximum %d", dataLen, maxLen)
	}

	return dataLen, nil
}

func (p *binaryVarPrefixer) Length() int {
	// Just when the field length is LLLL
	if p.Digits == 4 {
		return 2
	}
	return 1
}

func (p *binaryVarPrefixer) Inspect() string {
	return fmt.Sprintf("Binary.%s", strings.Repeat("L", p.Digits))
}
