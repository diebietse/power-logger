package logger

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClose(t *testing.T) {
	m := &mockModbus{
		readData: make([]byte, readSize*2),
	}
	l, err := New(m, "tester")
	assert.NoError(t, err, "Could not create logger")
	err = l.update()
	assert.NoError(t, err, "No update error expected")
	l.Poller()
	time.Sleep((pollRateSec + 1) * time.Second)
	l.Close()
}

func TestReadError(t *testing.T) {
	m := &mockModbus{
		readData: make([]byte, readSize*2),
		err:      errors.New("error"),
	}
	l, err := New(m, "tester-2")
	assert.NoError(t, err, "Could not create logger")
	err = l.update()
	assert.Error(t, err, "Error expected from update")
	l.Close()
}

func TestReadInvalidLength(t *testing.T) {
	m := &mockModbus{
		readData: make([]byte, 1),
	}
	l, err := New(m, "tester-3")
	assert.NoError(t, err, "Could not create logger")
	err = l.update()
	assert.Error(t, err, "Error expected from update")
	l.Close()
}

func TestGet16BitValue(t *testing.T) {
	v := get16BitValue([]byte{0x1, 0x10}, 0, 1)
	assert.InDelta(t, 272, v, 0.0001, "Value could not be extracted")
	v = get16BitValue([]byte{0x1, 0x10}, 0, 100)
	assert.InDelta(t, 2.72, v, 0.0001, "Value could not be extracted")
}

func TestGet32BitEnergy(t *testing.T) {
	v := get32BitEnergy([]byte{0x00, 0x1, 0x02, 0x10}, 0, 1)
	assert.InDelta(t, 66064, v, 0.0001, "Value could not be extracted")
	v = get32BitEnergy([]byte{0x00, 0x1, 0x02, 0x10}, 0, 100)
	assert.InDelta(t, 660.64, v, 0.0001, "Value could not be extracted")
}

type mockModbus struct {
	readData []byte
	err      error
}

func (m *mockModbus) ReadCoils(address, quantity uint16) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) ReadDiscreteInputs(address, quantity uint16) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) WriteSingleCoil(address, value uint16) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) WriteMultipleCoils(address, quantity uint16, value []byte) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) ReadInputRegisters(address, quantity uint16) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) ReadHoldingRegisters(address, quantity uint16) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) WriteSingleRegister(address, value uint16) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) WriteMultipleRegisters(address, quantity uint16, value []byte) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) MaskWriteRegister(address, andMask, orMask uint16) (results []byte, err error) {
	return m.readData, m.err
}
func (m *mockModbus) ReadFIFOQueue(address uint16) (results []byte, err error) {
	return m.readData, m.err
}
