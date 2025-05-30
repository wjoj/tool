package typesx

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wjoj/tool/v2/log"
)

type DecimalCalculateType int

const (
	DecimalCalculateTypeAdd DecimalCalculateType = 1 // *
	DecimalCalculateTypeSub DecimalCalculateType = 2 // -
	DecimalCalculateTypeMul DecimalCalculateType = 3 // *
	DecimalCalculateTypeDiv DecimalCalculateType = 4 // /
)

type Decimal struct {
	decimal.Decimal
}

func NewDecimal(value any) Decimal {
	dec := Decimal{}
	if err := dec.setvalue(value); err != nil {
		log.Warnf("decimal type conversion error %v", err)
	}
	return dec
}
func (d *Decimal) setvalue(value any) error {
	switch val := value.(type) {
	case int:
		d.Decimal = decimal.NewFromInt(int64(val))
	case int8:
		d.Decimal = decimal.NewFromInt(int64(val))
	case int16:
		d.Decimal = decimal.NewFromInt(int64(val))
	case uint:
		d.Decimal = decimal.NewFromUint64(uint64(val))
	case uint8:
		d.Decimal = decimal.NewFromUint64(uint64(val))
	case uint16:
		d.Decimal = decimal.NewFromUint64(uint64(val))
	case uint32:
		d.Decimal = decimal.NewFromUint64(uint64(val))
	case uint64:
		d.Decimal = decimal.NewFromUint64(val)
	case string:
		dec, err := decimal.NewFromString(val)
		if err != nil {
			return err
		}
		d.Decimal = dec
	case float64:
		d.Decimal = decimal.NewFromFloat(val)
	case float32:
		d.Decimal = decimal.NewFromFloat32(val)
	case int64:
		d.Decimal = decimal.NewFromInt(val)
	case int32:
		d.Decimal = decimal.NewFromInt32(val)
	case []byte:
		dec, err := decimal.NewFromString(string(val))
		if err != nil {
			return err
		}
		d.Decimal = dec
	case decimal.Decimal:
		d.Decimal = val
	case *decimal.Decimal:
		d.Decimal = *val
	case Decimal:
		d.Decimal = val.Decimal
	case *Decimal:
		d.Decimal = val.Decimal
	default:
		return fmt.Errorf("unsupported type %T", value)
	}
	return nil
}

// Add 在d值 + v
func (d *Decimal) Add(v any) {
	d.Decimal = d.Decimal.Add(NewDecimal(v).Decimal)
}

// NewAdd returns a new Add d + v
func (d Decimal) NewAdd(v any) Decimal {
	return Decimal{Decimal: d.Decimal.Add(NewDecimal(v).Decimal)}
}

// Sub 在d值 - v
func (d *Decimal) Sub(v any) {
	d.Decimal = d.Decimal.Sub(NewDecimal(v).Decimal)
}

// NewSub returns a new Sub d - v
func (d Decimal) NewSub(v any) Decimal {
	return Decimal{Decimal: d.Decimal.Sub(NewDecimal(v).Decimal)}
}

// Mul 在d值 * v
func (d *Decimal) Mul(v any) {
	d.Decimal = d.Decimal.Mul(NewDecimal(v).Decimal)
}

// NewMul returns a new Mul d * v
func (d Decimal) NewMul(v any) Decimal {
	return Decimal{Decimal: d.Decimal.Mul(NewDecimal(v).Decimal)}
}

// Div 在d值 / v
func (d *Decimal) Div(v any) {
	d.Decimal = d.Decimal.Div(NewDecimal(v).Decimal)
}

// NewDiv returns a new Div d / v
func (d Decimal) NewDiv(v any) Decimal {
	return Decimal{Decimal: d.Decimal.Div(NewDecimal(v).Decimal)}
}

// DivRound 在d值 / v
func (d *Decimal) DivRound(v any, precision int32) {
	d.Decimal = d.Decimal.DivRound(NewDecimal(v).Decimal, precision)
}

// NewDivRound returns a new Div d / v
func (d Decimal) NewDivRound(v any, precision int32) Decimal {
	return Decimal{Decimal: d.Decimal.DivRound(NewDecimal(v).Decimal, precision)}
}

// Calculator 数学计算
func (d *Decimal) Calculator(cty DecimalCalculateType, v any) error {
	if cty == DecimalCalculateTypeAdd {
		d.Add(v)
	} else if cty == DecimalCalculateTypeSub {
		d.Sub(v)
	} else if cty == DecimalCalculateTypeMul {
		d.Mul(v)
	} else if cty == DecimalCalculateTypeDiv {
		d.Div(v)
	} else {
		return errors.New("undefined calculation type")
	}
	return nil
}

// Calculator 数学计算 返回新值
func (d Decimal) NewCalculator(cty DecimalCalculateType, v any) Decimal {
	if cty == DecimalCalculateTypeAdd {
		return d.NewAdd(v)
	} else if cty == DecimalCalculateTypeSub {
		return d.NewSub(v)
	} else if cty == DecimalCalculateTypeMul {
		return d.NewMul(v)
	} else if cty == DecimalCalculateTypeDiv {
		return d.NewDiv(v)
	}
	return Decimal{}
}

func (d *Decimal) Scan(value any) error {
	return d.setvalue(value)
}

func (d Decimal) Value() (driver.Value, error) {
	return d.Decimal.String(), nil
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.Decimal.String()), nil
}

func (d *Decimal) UnmarshalJSON(data []byte) error {
	return d.setvalue(string(data))
}
