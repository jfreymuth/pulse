package main

import (
	"encoding/binary"
	"io"
	"os"
)

type File struct {
	out io.WriteSeeker
}

func CreateFile(name string, sampleRate, channels int) *File {
	file, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	return NewFile(file, sampleRate, channels)
}

func NewFile(w io.WriteSeeker, sampleRate, channels int) *File {
	io.WriteString(w, "RIFF    WAVEfmt ")
	var buf [20]byte
	binary.LittleEndian.PutUint32(buf[:], 16)
	binary.LittleEndian.PutUint16(buf[4:], 3)                              // format (3 = float)
	binary.LittleEndian.PutUint16(buf[6:], uint16(channels))               // channels
	binary.LittleEndian.PutUint32(buf[8:], uint32(sampleRate))             // sample rate
	binary.LittleEndian.PutUint32(buf[12:], uint32(sampleRate*4*channels)) // bytes/second
	binary.LittleEndian.PutUint16(buf[16:], uint16(4*channels))            // bytes/frame
	binary.LittleEndian.PutUint16(buf[18:], 32)                            // bits/sample
	w.Write(buf[:])
	io.WriteString(w, "data    ")
	return &File{out: w}
}

func (f *File) Write(p []float32) {
	binary.Write(f.out, binary.LittleEndian, p)
}

func (f *File) Close() {
	pos, _ := f.out.Seek(0, io.SeekCurrent)
	f.out.Seek(4, io.SeekStart)
	binary.Write(f.out, binary.LittleEndian, uint32(pos-8))
	f.out.Seek(40, io.SeekStart)
	binary.Write(f.out, binary.LittleEndian, uint32(pos-44))
	if c, ok := f.out.(io.Closer); ok {
		c.Close()
	}
}
