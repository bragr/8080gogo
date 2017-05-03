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
	// ------------------------------------------------------------------------
	case 0x41: // MOV B,C
		s.b = s.c
	case 0x42: // MOV B,D
		s.b = s.d
	case 0x43: // MOV B,E
		s.b = s.e
	// ------------------------------------------------------------------------
	case 0x76: //HALT
		os.Exit(0)
	// ------------------------------------------------------------------------
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
	case 0xc2: // JMZ adr
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
