package pricing

import "strconv"

type Decimal struct {
	offset   uint
	digits   []uint8
	perToken bool
}

func NewDecimal(offset uint, points ...uint8) Decimal {
	return Decimal{
		offset: offset, digits: points,
	}
}

func (d Decimal) PerToken() Decimal {
	d.perToken = true
	return d
}

func (d Decimal) String() string {
	b, _ := d.MarshalJSON()
	return string(b)
}

// Less reports whether d is numerically less than o. It ignores perToken,
// since the pricing command only sorts the per-million form.
func (d Decimal) Less(o Decimal) bool {
	aInt, aFrac := d.intFrac()
	bInt, bFrac := o.intFrac()
	if len(aInt) != len(bInt) {
		return len(aInt) < len(bInt)
	}
	for i := range aInt {
		if aInt[i] != bInt[i] {
			return aInt[i] < bInt[i]
		}
	}
	for i := 0; i < len(aFrac) && i < len(bFrac); i++ {
		if aFrac[i] != bFrac[i] {
			return aFrac[i] < bFrac[i]
		}
	}
	return len(aFrac) < len(bFrac)
}

func (d Decimal) intFrac() (intPart, fracPart []uint8) {
	n := len(d.digits)
	intLen := int(d.offset)
	if intLen > n {
		intLen = n
	}
	intPart = d.digits[:intLen]
	fracPart = d.digits[intLen:]
	for len(intPart) > 0 && intPart[0] == 0 {
		intPart = intPart[1:]
	}
	return intPart, fracPart
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	if len(d.digits) == 0 {
		return []byte{'0'}, nil
	}
	var b []byte
	if d.perToken {
		b = append(b, '0', '.')
		for range 6 - d.offset {
			b = append(b, '0')
		}
	} else if d.offset == 0 {
		b = append(b, '0', '.')
	}
	for i := range d.digits {
		if d.offset > 0 && uint(i) == d.offset && !d.perToken {
			b = append(b, '.')
		}
		s := strconv.FormatUint(uint64(d.digits[i]), 10)
		digit := []byte(s)[0]
		b = append(b, digit)
	}
	return b, nil
}

//  	if len(d.digits) == 0 {
//  		return []byte{'0'}, nil
//  	}
// -	b := []byte{'0', '.'}
// -	for range 6 - d.offset {
// -		b = append(b, '0')
// +	var b []byte
// +	if d.offset == 0 {
// +		b = append(b, '0', '.')
//  	}
//  	for i := range d.digits {
// +		if d.offset > 0 && uint(i) == d.offset {
// +			b = append(b, '.')
// +		}
//  		s := strconv.FormatUint(uint64(d.digits[i]), 10)
//  		digit := []byte(s)[0]
//  		b = append(b, digit)
