// Copyright (c) 2017, Grant A. Brady
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
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
	z   bool
	s   bool
	p   bool
	cy  bool
	ac  bool
	pad byte
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

func (s *State) getHL() uint16 {
	return (uint16(s.h) << 8) | uint16(s.l)
}

func (s *State) getCarry() uint16 {
	if s.cond.cy {
		return 1
	}
	return 0
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
	s.a = byte(answer & ROUND)
}

func (s *State) doZSPFlags(result uint8) {
	s.cond.z = (result == 0)
	s.cond.s = ((result & SIGN) != 0)
	s.parity(result)
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
		s.cond.cy = ((result & 0xffff0000) != 0)
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
	// ------------------------------------------------------------------------
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
	case 0x81: // ADD C
		answer := uint16(s.a) + uint16(s.c)
		s.doArithFlags(answer)
	case 0x82: // ADD D
		answer := uint16(s.a) + uint16(s.d)
		s.doArithFlags(answer)
	case 0x83: // ADD E
		answer := uint16(s.a) + uint16(s.e)
		s.doArithFlags(answer)
	case 0x84: // ADD H
		answer := uint16(s.a) + uint16(s.h)
		s.doArithFlags(answer)
	case 0x85: // ADD L
		answer := uint16(s.a) + uint16(s.l)
		s.doArithFlags(answer)
	case 0x86: // ADD M
		answer := uint16(s.a) + uint16(s.memory[s.getHL()])
		s.doArithFlags(answer)
	case 0x87: // ADD A
		answer := uint16(s.a) + uint16(s.a)
		s.doArithFlags(answer)
	case 0x88: // ADC B
		answer := uint16(s.a) + uint16(s.b) + s.getCarry()
		s.doArithFlags(answer)
	case 0x89: // ADC C
		answer := uint16(s.a) + uint16(s.c) + s.getCarry()
		s.doArithFlags(answer)
	case 0x8A: // ADC D
		answer := uint16(s.a) + uint16(s.d) + s.getCarry()
		s.doArithFlags(answer)
	case 0x8B: // ADC E
		answer := uint16(s.a) + uint16(s.e) + s.getCarry()
		s.doArithFlags(answer)
	case 0x8C: // ADC H
		answer := uint16(s.a) + uint16(s.h) + s.getCarry()
		s.doArithFlags(answer)
	case 0x8D: // ADC L
		answer := uint16(s.a) + uint16(s.l) + s.getCarry()
		s.doArithFlags(answer)
	case 0x8E: // ADC M
		answer := uint16(s.a) + uint16(s.memory[s.getHL()]) + s.getCarry()
		s.doArithFlags(answer)
	case 0x8F: // ADC A
		answer := uint16(s.a) + uint16(s.d) + s.getCarry()
		s.doArithFlags(answer)
	case 0x90: // SUB B
		answer := uint16(s.a) - uint16(s.b)
		s.doArithFlags(answer)
	case 0x91: // SUB C
		answer := uint16(s.a) - uint16(s.c)
		s.doArithFlags(answer)
	case 0x92: // SUB D
		answer := uint16(s.a) - uint16(s.d)
		s.doArithFlags(answer)
	case 0x93: // SUB E
		answer := uint16(s.a) - uint16(s.e)
		s.doArithFlags(answer)
	case 0x94: // SUB H
		answer := uint16(s.a) - uint16(s.h)
		s.doArithFlags(answer)
	case 0x95: // SUB L
		answer := uint16(s.a) - uint16(s.l)
		s.doArithFlags(answer)
	case 0x96: // SUB M
		answer := uint16(s.a) - uint16(s.memory[s.getHL()])
		s.doArithFlags(answer)
	case 0x97: // SUB A
		answer := uint16(s.a) - uint16(s.a)
		s.doArithFlags(answer)
	case 0x98: // SBB B
		answer := uint16(s.a) - uint16(s.b) - s.getCarry()
		s.doArithFlags(answer)
	case 0x99: // SBB C
		answer := uint16(s.a) - uint16(s.c) - s.getCarry()
		s.doArithFlags(answer)
	case 0x9A: // SBB D
		answer := uint16(s.a) - uint16(s.d) - s.getCarry()
		s.doArithFlags(answer)
	case 0x9B: // SBB E
		answer := uint16(s.a) - uint16(s.e) - s.getCarry()
		s.doArithFlags(answer)
	case 0x9C: // SBB H
		answer := uint16(s.a) - uint16(s.h) - s.getCarry()
		s.doArithFlags(answer)
	case 0x9D: // SBB L
		answer := uint16(s.a) - uint16(s.l) - s.getCarry()
		s.doArithFlags(answer)
	case 0x9E: // SBB M
		answer := uint16(s.a) - uint16(s.memory[s.getHL()]) - s.getCarry()
		s.doArithFlags(answer)
	case 0x9F: // SBB A
		answer := uint16(s.a) - uint16(s.d) - s.getCarry()
		s.doArithFlags(answer)
	// ------------------------------------------------------------------------
	case 0xc2: // JNZ adr
		if !s.cond.z {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xc3: // JMP adr
		s.pc = s.getAddr()
	case 0xc6:
		s.pc++
		answer := uint16(s.a) + uint16(s.memory[s.pc])
		s.doArithFlags(answer)
	case 0xca: // JZ adr
		if s.cond.z {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xd2: // JNC adr
		if !s.cond.cy {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xda: // JC adr
		if s.cond.cy {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xe2: // JPO adr
		if !s.cond.p {
			s.pc = s.getAddr()
		} else {
			s.pc += 1
		}
	case 0xea: // JPE adr
		if s.cond.p {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
	case 0xf2: // JP adr
		if !s.cond.s {
			s.pc = s.getAddr()
		} else {
			s.pc += 2
		}
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
