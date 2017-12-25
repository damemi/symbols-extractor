package contracts

import (
	"fmt"
	"go/token"

	"github.com/gofed/symbols-extractor/pkg/parser/contracts/typevars"
	gotypes "github.com/gofed/symbols-extractor/pkg/types"
)

type Type string

type Contract interface {
	GetType() Type
}

var BinaryOpType Type = "binaryop"
var UnaryOpType Type = "unaryop"
var PropagatesToType Type = "propagatesto"
var IsCompatibleWithType Type = "iscompatiblewith"
var IsInvocableType Type = "isinvocable"
var HasFieldType Type = "hasfield"
var IsReferenceableType Type = "isreferenceable"
var ReferenceOfType Type = "referenceof"
var IsDereferenceableType Type = "Isdereferenceable"
var DereferenceOfType Type = "dereferenceOf"
var IsIndexableType Type = "isindexable"
var IsSendableToType Type = "issendableto"
var IsReceiveableFromType Type = "isreceiveablefrom"
var IsIncDecableType Type = "isincdecable"
var IsRangeableType Type = "israngeable"

func Contract2String(c Contract) string {
	switch d := c.(type) {
	case *BinaryOp:
		return fmt.Sprintf("BinaryOpContract:\n\tX=%v,\n\tY=%v,\n\tZ=%v,\n\top=%v", typevars.TypeVar2String(d.X), typevars.TypeVar2String(d.Y), typevars.TypeVar2String(d.Z), d.OpToken)
	case *UnaryOp:
		return fmt.Sprintf("UnaryOpContract:\n\tX=%v,\n\tY=%v,\n\top=%v", typevars.TypeVar2String(d.X), typevars.TypeVar2String(d.Y), d.OpToken)
	case *PropagatesTo:
		return fmt.Sprintf("PropagatesTo:\n\tX=%v,\n\tY=%v,\n\tE=%#v", typevars.TypeVar2String(d.X), typevars.TypeVar2String(d.Y), d.ExpectedType)
	case *IsCompatibleWith:
		return fmt.Sprintf("IsCompatibleWith:\n\tX=%v\n\tY=%v\n\tWeak=%v\n\tE=%v", typevars.TypeVar2String(d.X), typevars.TypeVar2String(d.Y), d.Weak, d.ExpectedType)
	case *IsInvocable:
		return fmt.Sprintf("IsInvocable:\n\tF=%v,\n\targCount=%v", typevars.TypeVar2String(d.F), d.ArgsCount)
	case *IsReferenceable:
		return fmt.Sprintf("IsReferenceable:\n\tX=%v", typevars.TypeVar2String(d.X))
	case *IsDereferenceable:
		return fmt.Sprintf("IsDereferenceable:\n\tX=%v", typevars.TypeVar2String(d.X))
	case *ReferenceOf:
		return fmt.Sprintf("ReferenceOf:\n\tX=%v,\n\tY=%v", typevars.TypeVar2String(d.X), typevars.TypeVar2String(d.Y))
	case *DereferenceOf:
		return fmt.Sprintf("DereferenceOf:\n\tX=%v,\n\tY=%v", typevars.TypeVar2String(d.X), typevars.TypeVar2String(d.Y))
	case *HasField:
		return fmt.Sprintf("HasField:\n\tX=%v,\n\tField=%v,\n\tIndex=%v", typevars.TypeVar2String(d.X), d.Field, d.Index)
	case *IsIndexable:
		return fmt.Sprintf("IsIndexable:\n\tX=%v", typevars.TypeVar2String(d.X))
	case *IsSendableTo:
		return fmt.Sprintf("IsSendableTo:\n\tX=%v\n\tY=%v", typevars.TypeVar2String(d.X), typevars.TypeVar2String(d.Y))
	case *IsReceiveableFrom:
		return fmt.Sprintf("IsReceiveableFrom:\n\tX=%v\n\tY=%v", typevars.TypeVar2String(d.X), typevars.TypeVar2String(d.Y))
	case *IsIncDecable:
		return fmt.Sprintf("IsIncDecable:\n\tX=%v", typevars.TypeVar2String(d.X))
	case *IsRangeable:
		return fmt.Sprintf("IsRangeable:\n\tX=%v", typevars.TypeVar2String(d.X))
	default:
		panic(fmt.Sprintf("Contract %#v not recognized", c))
	}
}

// BinaryOp represents contract between two typevars
type BinaryOp struct {
	// OpToken gives information about particular binary operation.
	// E.g. '+' can be used with integers and strings, '-' can not be used with strings.
	// As long as the operands are compatible with the operation, the contract holds.
	OpToken token.Token
	// Z = X op Y
	X, Y, Z      typevars.Interface
	ExpectedType gotypes.DataType
}

type UnaryOp struct {
	OpToken token.Token
	// Y = op X
	X, Y         typevars.Interface
	ExpectedType gotypes.DataType
}

type PropagatesTo struct {
	X, Y         typevars.Interface
	ExpectedType gotypes.DataType
}

type IsCompatibleWith struct {
	X, Y         typevars.Interface
	ExpectedType gotypes.DataType
	// As long as MapKey is compatible with integer, it is compatible with ListKey as well
	// TODO(jchaloup): make sure this principle is applied during the compatibility analysis
	Weak bool
}

type IsInvocable struct {
	F         typevars.Interface
	ArgsCount int
}

type HasField struct {
	X     typevars.Interface
	Field string
	Index int
}

type IsReferenceable struct {
	X typevars.Interface
}

type ReferenceOf struct {
	X, Y typevars.Interface
}

type IsDereferenceable struct {
	X typevars.Interface
}

type DereferenceOf struct {
	X, Y typevars.Interface
}

type IsIndexable struct {
	X, Key  typevars.Interface
	IsSlice bool
}

type IsSendableTo struct {
	X, Y typevars.Interface
}

type IsReceiveableFrom struct {
	X, Y         typevars.Interface
	ExpectedType gotypes.DataType
}

type IsIncDecable struct {
	X typevars.Interface
}

type IsRangeable struct {
	X typevars.Interface
}

func (b *BinaryOp) GetType() Type {
	return BinaryOpType
}

func (b *UnaryOp) GetType() Type {
	return UnaryOpType
}

func (p *PropagatesTo) GetType() Type {
	return PropagatesToType
}

func (i *IsCompatibleWith) GetType() Type {
	return IsCompatibleWithType
}

func (i *IsInvocable) GetType() Type {
	return IsInvocableType
}

func (i *HasField) GetType() Type {
	return HasFieldType
}

func (i *IsReferenceable) GetType() Type {
	return IsReferenceableType
}

func (i *IsDereferenceable) GetType() Type {
	return IsDereferenceableType
}

func (i *ReferenceOf) GetType() Type {
	return ReferenceOfType
}

func (i *DereferenceOf) GetType() Type {
	return DereferenceOfType
}

func (i *IsIndexable) GetType() Type {
	return IsIndexableType
}

func (i *IsSendableTo) GetType() Type {
	return IsSendableToType
}

func (i *IsReceiveableFrom) GetType() Type {
	return IsReceiveableFromType
}

func (i *IsIncDecable) GetType() Type {
	return IsIncDecableType
}

func (i *IsRangeable) GetType() Type {
	return IsRangeableType
}
