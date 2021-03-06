// Copyright (c) 2017, Grant A. Brady
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
)

const (
	ZERO  uint16 = 0xFF
	SIGN  uint16 = 0x80
	CARRY uint16 = 0xFF
	ROUND uint16 = 0xFF
	ROM   int    = 0x2000
)

var IN_FILE string

type Conditions struct {
	cy   bool
	pad1 bool
	p    bool
	pad2 bool
	ac   bool
	pad3 bool
	z    bool
	s    bool
}

type State struct {
	a      byte
	b      byte
	c      byte
	d      byte
	e      byte
	h      byte
	l      byte
	sp     uint16
	pc     uint16
	memory [0x10000]byte
	cond   Conditions
	enable byte
}

func (s *State) getBC() uint16 {
	return (uint16(s.b) << 8) | uint16(s.c)
}

func (s *State) setBC(bc uint16) {
	s.b = uint8((bc * 0xff00) >> 8)
	s.c = uint8(bc & 0xff)
}

func (s *State) getDE() uint16 {
	return (uint16(s.d) << 8) | uint16(s.e)
}

func (s *State) setDE(de uint16) {
	s.e = uint8((de * 0xff00) >> 8)
	s.d = uint8(de & 0xff)
}

func (s *State) getHL() uint16 {
	return (uint16(s.h) << 8) | uint16(s.l)
}

func (s *State) setHL(hl uint16) {
	s.l = uint8((hl * 0xff00) >> 8)
	s.h = uint8(hl & 0xff)
}

func (s *State) getCarry() uint16 {
	if s.cond.cy {
		return 1
	}
	return 0
}

func (s *State) setCarry(result uint32) {
	s.cond.cy = ((result & 0xffff0000) != 0)
}

func (s *State) getAddr() uint16 {
	return (uint16(s.memory[s.pc+2]) << 8) | uint16(s.memory[s.pc+1])
}

func (s *State) unimplemented() {
	opcode := s.memory[s.pc]
	fmt.Printf("Unimplemented: 0x%02x\n", opcode)
	os.Exit(1)
}

func (s *State) parity(answer byte) {
	const SIZE byte = 8
	var p byte = 0
	answer = answer & ((1 << SIZE) - 1)
	for i := byte(0); i < SIZE; i++ {
		if (answer & 0x1) > 0 {
			p++
		}
		answer = answer >> 1
	}
	s.cond.p = (p & 0x1) == 0
}

func (s *State) doArithFlags(answer uint16) {
	s.cond.z = ((answer & ZERO) == 0)
	s.cond.s = ((answer & SIGN) != 0)
	s.cond.cy = (answer > CARRY)
	s.parity(byte(answer))
}

func (s *State) doLogicFlags() {
	s.cond.z = (s.a == 0)
	s.cond.s = ((s.a & uint8(SIGN)) != 0)
	s.cond.cy = !s.cond.ac
	s.parity(s.a)
}

func (s *State) doZSPFlags(result uint8) {
	s.cond.z = (result == 0)
	s.cond.s = ((result & uint8(SIGN)) != 0)
	s.parity(result)
}

func (s *State) push(high, low uint8) {
	s.memory[s.sp-1] = high
	s.memory[s.sp-2] = low
	s.sp -= 2
}

func (s *State) pop() (high, low uint8) {
	low = s.memory[s.sp]
	high = s.memory[s.sp+1]
	s.sp += 2
	return
}

func (s *State) Emulate() {
	opcode := s.memory[s.pc]
	fmt.Printf("%06d: 0x%02x\n", s.pc, opcode)

	switch opcode {
	case 0x00: // NOP
		break
	case 0x01: // LXI B,word
		s.pc++
		s.c = s.memory[s.pc]
		s.pc++
		s.b = s.memory[s.pc]
	case 0x02: // STAX B
		s.memory[s.getBC()] = s.a
	case 0x03: // INX B
		s.c++
		if s.c == 0 {
			s.b++
		}
	case 0x04: // INR B
		s.b++
		s.doZSPFlags(s.b)
	case 0x05: // DCR B
		s.b--
		s.doZSPFlags(s.b)
	case 0x06: // MVI B, D8
		s.pc++
		s.b = s.memory[s.pc]
	case 0x07: // RLC
		bit7 := s.a & 0x80
		s.a = (s.a << 1) | (bit7 >> 7)
		s.cond.cy = bit7 == 0x80
	case 0x08: // Unimplemented
		s.unimplemented()
	case 0x09: // DAD B
		result := uint32(s.getHL()) + uint32(s.getBC())
		s.h = uint8((result * 0xff00) >> 8)
		s.l = uint8(result & 0xff)
		s.setCarry(result)
	case 0x0a: // LDAX B
		s.a = s.memory[s.getBC()]
	case 0x0b: // DCX B
		s.setBC(s.getBC() - 1)
	case 0x0c: // INR C
		s.c++
		s.doZSPFlags(s.c)
	case 0x0d: // DCR C
		s.c--
		s.doZSPFlags(s.c)
	case 0x0e: // MVI C, D8
		s.pc++
		s.c = s.memory[s.pc]
	case 0x0f: // RRC
		bit0 := s.a & 0x01
		s.a = (s.a >> 1) | (bit0 << 7)
		s.cond.cy = (bit0 == 0x01)
	case 0x10: // Unimplemented
		s.unimplemented()
	case 0x11: // LXI D, word
		s.pc++
		s.e = s.memory[s.pc]
		s.pc++
		s.d = s.memory[s.pc]
	case 0x12: // STAX D
		s.memory[s.getDE()] = s.a
	case 0x13: // INX D
		s.memory[s.getDE()] = s.a
	case 0x14: // INR D
		s.d++
		s.doZSPFlags(s.d)
	case 0x15: // DCR D
		s.d--
		s.doZSPFlags(s.d)
	case 0x16: // MVI D, D8
		s.pc++
		s.d = s.memory[s.pc]
	case 0x17: // RAL
		bit7 := s.a & 0x80
		if s.cond.cy {
			s.a = s.a | 0x01
		}
		s.cond.cy = bit7 == 0x80
	case 0x18: // Unimplemented
		s.unimplemented()
	case 0x19: // DAD D
		result := uint32(s.getHL()) + uint32(s.getDE())
		s.h = uint8((result * 0xff00) >> 8)
		s.l = uint8(result & 0xff)
		s.setCarry(result)
	case 0x1a: // LDAX D
		s.a = s.memory[s.getDE()]
	case 0x1b: // DCX D
		s.setDE(s.getDE() - 1)
	case 0x1c: // INR E
		s.e++
		s.doZSPFlags(s.e)
	case 0x1d: // DCR E
		s.e--
		s.doZSPFlags(s.e)
	case 0x1e: // MVI E,D8
		s.pc++
		s.e = s.memory[s.pc]
	case 0x1f: // RAR
		s.cond.cy = (s.a & 0x01) == 0x01
		bit7 := s.a & 0x80
		s.a = (s.a >> 1) | bit7
	case 0x20: // RIM
		s.unimplemented()
	case 0x21: // LXI H, D16
		s.pc++
		s.l = s.memory[s.pc]
		s.pc++
		s.h = s.memory[s.pc]
	case 0x22: // SHLD adr
		adr := s.getAddr()
		s.pc += 2
		s.memory[adr] = s.l
		s.memory[adr+1] = s.h
	case 0x23: // INX H
		s.setHL(s.getHL() + 1)
	case 0x24: // INR H
		s.h++
		s.doZSPFlags(s.h)
	case 0x25: // DCR H
		s.h--
		s.doZSPFlags(s.h)
	case 0x26: // MVI H,D8
		s.pc++
		s.h = s.memory[s.pc]
	case 0x27: // DAA
		s.unimplemented()
	case 0x28: // Unimplemented
		s.unimplemented()
	case 0x29: // DAD H
		result := 2 * uint32(s.getHL())
		s.setHL(uint16(result))
		s.setCarry(result)
	case 0x2a: // LHLD adr
		adr := s.getAddr()
		s.pc += 2
		s.l = s.memory[adr]
		s.h = s.memory[adr+1]
	case 0x2b: // DCX H
		s.setHL(s.getHL() - 1)
	case 0x2c: // INR L
		s.l++
		s.doZSPFlags(s.l)
	case 0x2d: // DCR L
		s.l--
		s.doZSPFlags(s.l)
	case 0x2f: // CMA
		s.a = ^s.a
	case 0x30: // SIM
		s.unimplemented()
	case 0x31: // LXI SP,D16
		s.sp = s.getAddr()
		s.pc += 2
	case 0x32: // STA adr
		s.memory[s.getAddr()] = s.a
		s.pc++
	case 0x33: // INX SP
		s.sp = s.sp + 1
	case 0x34: // INR M
		hl := s.getHL()
		s.memory[hl] = s.memory[hl] + 1
		s.doZSPFlags(s.memory[hl])
	case 0x35: // DCR M
		hl := s.getHL()
		s.memory[hl] = s.memory[hl] - 1
		s.doZSPFlags(s.memory[hl])
	case 0x36: // MVI M,D8
		s.pc++
		s.memory[s.getHL()] = s.memory[s.pc]
	case 0x37: // STC
		s.cond.cy = true
	case 0x38: // Unimplemented
		s.unimplemented()
	case 0x39: // DAD SP
		result := uint32(s.getHL()) + uint32(s.sp)
		s.setHL(uint16(result))
		s.setCarry(result)
	case 0x3a: // LDA adr
		s.a = s.memory[s.getAddr()]
		s.pc += 2
	case 0x3b: // DCX SP
		s.sp = s.sp - 1
	case 0x3c: // INR A
		s.a++
		s.doZSPFlags(s.a)
	case 0x3d: // DCR A
		s.a--
		s.doZSPFlags(s.a)
	case 0x3f: // CMC
		s.cond.cy = !s.cond.cy
	case 0x40: // MOV B,b
		s.b = s.b
	case 0x41: // MOV B,C
		s.b = s.c
	case 0x42: // MOV B,D
		s.b = s.d
	case 0x43: // MOV B,E
		s.b = s.e
	case 0x44: // MOV B,H
		s.b = s.h
	case 0x45: // MOV B,L
		s.b = s.l
	case 0x46: // MOV B,M
		s.b = s.memory[s.getAddr()]
	case 0x47: // MOV B,A
		s.b = s.a
	case 0x48: // MOV C,B
		s.c = s.b
	case 0x49: // MOV C,C
		s.c = s.c
	case 0x4a: // MOV C,D
		s.c = s.d
	case 0x4b: // MOV C,E
		s.c = s.e
	case 0x4c: // MOV C,H
		s.c = s.h
	case 0x4d: // MOV C,L
		s.c = s.l
	case 0x4e: // MOV C,M
		s.c = s.memory[s.getAddr()]
	case 0x4f: // MOV C,A
		s.c = s.a
	case 0x50: // MOV D,B
		s.d = s.b
	case 0x51: // MOV D,C
		s.d = s.c
	case 0x52: // MOV D,D
		s.d = s.d
	case 0x53: // MOV D,E
		s.d = s.e
	case 0x54: // MOV D,H
		s.d = s.h
	case 0x55: // MOV D,L
		s.d = s.l
	case 0x56: // MOV D,M
		s.d = s.memory[s.getAddr()]
	case 0x57: // MOV D,A
		s.d = s.a
	case 0x58: // MOV E,B
		s.e = s.b
	case 0x59: // MOV E,C
		s.e = s.c
	case 0x5a: // MOV E,D
		s.e = s.d
	case 0x5b: // MOV E,E
		s.e = s.e
	case 0x5c: // MOV E,H
		s.e = s.h
	case 0x5d: // MOV E,L
		s.e = s.l
	case 0x5e: // MOV E,M
		s.e = s.memory[s.getAddr()]
	case 0x5f: // MOV E,A
		s.e = s.a
	case 0x60: // MOV H,B
		s.h = s.b
	case 0x61: // MOV H,C
		s.h = s.c
	case 0x62: // MOV H,D
		s.h = s.d
	case 0x63: // MOV H,E
		s.h = s.e
	case 0x64: // MOV H,H
		s.h = s.h
	case 0x65: // MOV H,L
		s.h = s.l
	case 0x66: // MOV H,M
		s.h = s.memory[s.getAddr()]
	case 0x67: // MOV H,A
		s.h = s.a
	case 0x68: // MOV L,B
		s.l = s.b
	case 0x69: // MOV L,C
		s.l = s.c
	case 0x6a: // MOV L,D
		s.l = s.d
	case 0x6b: // MOV L,E
		s.l = s.e
	case 0x6c: // MOV L,H
		s.l = s.h
	case 0x6d: // MOV L,L
		s.l = s.l
	case 0x6e: // MOV L,M
		s.l = s.memory[s.getAddr()]
	case 0x6f: // MOV L,A
		s.l = s.a
	case 0x70: // MOV M,B
		s.memory[s.getAddr()] = s.b
	case 0x71: // MOV M,C
		s.memory[s.getAddr()] = s.c
	case 0x72: // MOV M,D
		s.memory[s.getAddr()] = s.d
	case 0x73: // MOV M,E
		s.memory[s.getAddr()] = s.e
	case 0x74: // MOV M,H
		s.memory[s.getAddr()] = s.h
	case 0x75: // MOV M,L
		s.memory[s.getAddr()] = s.l
	case 0x76: //HALT
		os.Exit(0)
	case 0x77: // MOV M,A
		s.memory[s.getAddr()] = s.a
	case 0x78: // MOV A,B
		s.a = s.b
	case 0x79: // MOV A,C
		s.a = s.c
	case 0x7a: // MOV A,D
		s.a = s.d
	case 0x7b: // MOV A,E
		s.a = s.e
	case 0x7c: // MOV A,H
		s.a = s.h
	case 0x7d: // MOV A,L
		s.a = s.l
	case 0x7e: // MOV A,M
		s.a = s.memory[s.getAddr()]
	case 0x7f: // MOV A,A
		s.a = s.a
	case 0x80: // ADD B
		answer := uint16(s.a) + uint16(s.b)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x81: // ADD C
		answer := uint16(s.a) + uint16(s.c)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x82: // ADD D
		answer := uint16(s.a) + uint16(s.d)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x83: // ADD E
		answer := uint16(s.a) + uint16(s.e)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x84: // ADD H
		answer := uint16(s.a) + uint16(s.h)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x85: // ADD L
		answer := uint16(s.a) + uint16(s.l)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x86: // ADD M
		answer := uint16(s.a) + uint16(s.memory[s.getHL()])
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x87: // ADD A
		answer := uint16(s.a) + uint16(s.a)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x88: // ADC B
		answer := uint16(s.a) + uint16(s.b) + s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x89: // ADC C
		answer := uint16(s.a) + uint16(s.c) + s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x8A: // ADC D
		answer := uint16(s.a) + uint16(s.d) + s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x8B: // ADC E
		answer := uint16(s.a) + uint16(s.e) + s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x8C: // ADC H
		answer := uint16(s.a) + uint16(s.h) + s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x8D: // ADC L
		answer := uint16(s.a) + uint16(s.l) + s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x8E: // ADC M
		answer := uint16(s.a) + uint16(s.memory[s.getHL()]) + s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x8F: // ADC A
		answer := uint16(s.a) + uint16(s.d) + s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x90: // SUB B
		answer := uint16(s.a) - uint16(s.b)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x91: // SUB C
		answer := uint16(s.a) - uint16(s.c)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x92: // SUB D
		answer := uint16(s.a) - uint16(s.d)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x93: // SUB E
		answer := uint16(s.a) - uint16(s.e)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x94: // SUB H
		answer := uint16(s.a) - uint16(s.h)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x95: // SUB L
		answer := uint16(s.a) - uint16(s.l)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x96: // SUB M
		answer := uint16(s.a) - uint16(s.memory[s.getHL()])
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x97: // SUB A
		answer := uint16(s.a) - uint16(s.a)
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x98: // SBB B
		answer := uint16(s.a) - uint16(s.b) - s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x99: // SBB C
		answer := uint16(s.a) - uint16(s.c) - s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x9A: // SBB D
		answer := uint16(s.a) - uint16(s.d) - s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x9B: // SBB E
		answer := uint16(s.a) - uint16(s.e) - s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x9C: // SBB H
		answer := uint16(s.a) - uint16(s.h) - s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x9D: // SBB L
		answer := uint16(s.a) - uint16(s.l) - s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x9E: // SBB M
		answer := uint16(s.a) - uint16(s.memory[s.getHL()]) - s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0x9F: // SBB A
		answer := uint16(s.a) - uint16(s.d) - s.getCarry()
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0xa0: // ANA B
		s.a = s.a & s.b
		s.doLogicFlags()
	case 0xa1: // ANA c
		s.a = s.a & s.c
		s.doLogicFlags()
	case 0xa2: // ANA D
		s.a = s.a & s.d
		s.doLogicFlags()
	case 0xa3: // ANA E
		s.a = s.a & s.e
		s.doLogicFlags()
	case 0xa4: // ANA H
		s.a = s.a & s.h
		s.doLogicFlags()
	case 0xa5: // ANA L
		s.a = s.a & s.l
		s.doLogicFlags()
	case 0xa6: // ANA M
		s.a = s.a & s.memory[s.getCarry()]
		s.pc += 2
		s.doLogicFlags()
	case 0xa7: // ANA A
		s.a = s.a & s.a
		s.doLogicFlags()
	case 0xa8: // XRA B
		s.a = s.a ^ s.b
		s.doLogicFlags()
	case 0xa9: // XRA C
		s.a = s.a ^ s.c
		s.doLogicFlags()
	case 0xaa: // XRA D
		s.a = s.a ^ s.d
		s.doLogicFlags()
	case 0xab: // XRA E
		s.a = s.a ^ s.e
		s.doLogicFlags()
	case 0xac: // XRA H
		s.a = s.a ^ s.h
		s.doLogicFlags()
	case 0xad: // XRA l
		s.a = s.a ^ s.l
		s.doLogicFlags()
	case 0xae: // XRA M
		s.a = s.a ^ s.memory[s.getAddr()]
		s.pc += 2
		s.doLogicFlags()
	case 0xaf: // XRA A
		s.a = s.a ^ s.a
		s.doLogicFlags()
	case 0xb0: // ORA B
		s.a = s.a | s.b
		s.doLogicFlags()
	case 0xb1: // ORA C
		s.a = s.a | s.c
		s.doLogicFlags()
	case 0xb2: // ORA D
		s.a = s.a | s.d
		s.doLogicFlags()
	case 0xb3: // ORA E
		s.a = s.a | s.e
		s.doLogicFlags()
	case 0xb4: // ORA H
		s.a = s.a | s.h
		s.doLogicFlags()
	case 0xb5: // ORA L
		s.a = s.a | s.l
		s.doLogicFlags()
	case 0xb6: // ORA M
		s.a = s.a | s.memory[s.getAddr()]
		s.pc += 2
		s.doLogicFlags()
	case 0xb7: // ORA a
		s.a = s.a | s.a
		s.doLogicFlags()
	case 0xb8: // CMP B
		s.doArithFlags(uint16(s.a) - uint16(s.b))
	case 0xb9: // CMP C
		s.doArithFlags(uint16(s.a) - uint16(s.c))
	case 0xba: // CMP D
		s.doArithFlags(uint16(s.a) - uint16(s.d))
	case 0xbb: // CMP E
		s.doArithFlags(uint16(s.a) - uint16(s.e))
	case 0xbc: // CMP H
		s.doArithFlags(uint16(s.a) - uint16(s.h))
	case 0xbd: // CMP L
		s.doArithFlags(uint16(s.a) - uint16(s.l))
	case 0xbe: // CMP M
		s.doArithFlags(uint16(s.a) - uint16(s.memory[s.getAddr()]))
		s.pc += 2
	case 0xbf: // CMP A
		s.doArithFlags(uint16(s.a) - uint16(s.a))
	case 0xc1: // POP B
		s.b, s.c = s.pop()
	case 0xc2: // JNZ adr
		if !s.cond.z {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xc3: // JMP adr
		s.pc = s.getAddr()
	case 0xc5: // PUSH B
		s.push(s.b, s.c)
	case 0xc6:
		s.pc++
		answer := uint16(s.a) + uint16(s.memory[s.pc])
		s.doArithFlags(answer)
		s.a = byte(answer & ROUND)
	case 0xca: // JZ adr
		if s.cond.z {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xd1: // POP D
		s.d, s.e = s.pop()
	case 0xd2: // JNC adr
		if !s.cond.cy {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xd5: // PUSH D
		s.push(s.d, s.e)
	case 0xda: // JC adr
		if s.cond.cy {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xe1: // POP H
		s.h, s.l = s.pop()
	case 0xe2: // JPO adr
		if !s.cond.p {
			s.pc = s.getAddr()
		} else {
			s.pc += 1
		}
	case 0xe5: // PUSH H
		s.push(s.h, s.l)
	case 0xea: // JPE adr
		if s.cond.p {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xf1: // POP PWS
		var low uint8
		s.a, low = s.pop()
		s.cond.cy = (low & 0x01) == 0x01
		s.cond.pad1 = (low & 0x02) == 0x02
		s.cond.p = (low & 0x04) == 0x04
		s.cond.pad2 = (low & 0x08) == 0x08
		s.cond.ac = (low & 0x10) == 0x10
		s.cond.pad3 = (low & 0x20) == 0x20
		s.cond.z = (low & 0x40) == 0x40
		s.cond.s = (low & 0x80) == 0x80
	case 0xf2: // JP adr
		if !s.cond.s {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xf5: // PUSH PSW
		var psw uint8 = 0
		var flag uint8 = 0x01
		v := reflect.ValueOf(s.cond)
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).Bool() {
				psw = psw | flag
			}
			flag = flag << 1
		}

		s.push(s.a, psw)
	case 0xfa: // JM adr
		if s.cond.s {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	default:
		s.unimplemented()
	}
	s.pc += 1
}

func init() {
	flag.StringVar(&IN_FILE, "ROM", "in.rom", "File to read as rom/ram")
	flag.Parse()
}

func main() {
	fmt.Println("Hello 8080gogo")
	defer fmt.Println("Goodbye 8080gogo")
	fmt.Println(IN_FILE)

	f, err := os.Open(IN_FILE)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	d, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	f.Close()
	if len(d) == 0 {
		fmt.Println("Zero size ROM. Abort")
		os.Exit(1)
	}

	state := new(State)
	copy(state.memory[:], d)

	fmt.Println("Emulator startup!")
	for {
		state.Emulate()
	}
}
