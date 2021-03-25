package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const mfID = 1

func TestExactMatch(t *testing.T) {
	testCases := []struct {
		canonical bool
		in        []byte
		out       []byte
	}{
		{true, []byte{'\x00', '\xab'}, []byte{'\xab'}},
		{false, []byte{'\x00', '\xab'}, []byte{'\x00', '\xab'}},
	}

	for _, tc := range testCases {
		m := ExactMatch{Value: tc.in}
		mf := m.get(mfID, tc.canonical)
		assert.Equal(t, tc.out, mf.GetExact().Value)
	}
}

func TestLPMMatch(t *testing.T) {
	testCases := []struct {
		canonical bool
		in        []byte
		pLen      int32
		out       []byte
	}{
		{true, []byte{'\x00', '\xab'}, 16, []byte{'\xab'}},
		{false, []byte{'\x00', '\xab'}, 16, []byte{'\x00', '\xab'}},
		{true, []byte{'\x00', '\xab'}, 8, []byte{'\x00'}},
	}

	for _, tc := range testCases {
		m := LpmMatch{Value: tc.in, PLen: tc.pLen}
		mf := m.get(mfID, tc.canonical)
		assert.Equal(t, tc.out, mf.GetLpm().Value)
		assert.Equal(t, tc.pLen, mf.GetLpm().PrefixLen)
	}
}

func TestTernaryMatch(t *testing.T) {
	testCases := []struct {
		canonical bool
		valueIn   []byte
		maskIn    []byte
		valueOut  []byte
		maskOut   []byte
	}{
		{true, []byte{'\x00', '\xab'}, []byte{'\x00', '\xf0'}, []byte{'\xa0'}, []byte{'\xf0'}},
		{false, []byte{'\x00', '\xab'}, []byte{'\x00', '\xf0'}, []byte{'\x00', '\xa0'}, []byte{'\x00', '\xf0'}},
		{true, []byte{'\xab', '\x00'}, []byte{'\xff'}, []byte{'\x00'}, []byte{'\xff'}},
		{true, []byte{'\xab'}, []byte{'\x0f', '\x0f'}, []byte{'\x0b'}, []byte{'\x0f', '\x0f'}},
	}

	for _, tc := range testCases {
		m := TernaryMatch{Value: tc.valueIn, Mask: tc.maskIn}
		mf := m.get(mfID, tc.canonical)
		assert.Equal(t, tc.valueOut, mf.GetTernary().Value)
		assert.Equal(t, tc.maskOut, mf.GetTernary().Mask)
	}
}

func TestRangeMatch(t *testing.T) {
	testCases := []struct {
		canonical bool
		lowIn     []byte
		highIn    []byte
		lowOut    []byte
		highOut   []byte
	}{
		{true, []byte{'\x00', '\xab'}, []byte{'\xff', '\xf0'}, []byte{'\xab'}, []byte{'\xff', '\xf0'}},
		{false, []byte{'\x00', '\xab'}, []byte{'\xff', '\xf0'}, []byte{'\x00', '\xab'}, []byte{'\xff', '\xf0'}},
	}

	for _, tc := range testCases {
		m := RangeMatch{Low: tc.lowIn, High: tc.highIn}
		mf := m.get(mfID, tc.canonical)
		assert.Equal(t, tc.lowOut, mf.GetRange().Low)
		assert.Equal(t, tc.highOut, mf.GetRange().High)
	}
}

func TestOptionalMatch(t *testing.T) {
	testCases := []struct {
		canonical bool
		in        []byte
		out       []byte
	}{
		{true, []byte{'\x00', '\xab'}, []byte{'\xab'}},
		{false, []byte{'\x00', '\xab'}, []byte{'\x00', '\xab'}},
	}

	for _, tc := range testCases {
		m := OptionalMatch{Value: tc.in}
		mf := m.get(mfID, tc.canonical)
		assert.Equal(t, tc.out, mf.GetOptional().Value)
	}
}
