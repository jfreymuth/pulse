package main

import (
	"fmt"
	"os"

	"github.com/jfreymuth/pulse"
)

func main() {
	c, err := pulse.NewClient()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	file := CreateFile("out.wav", 44100, 1)
	stream, err := c.NewRecord(file.Write)
	if err != nil {
		fmt.Println(err)
		return
	}

	stream.Start()

	fmt.Print("Press enter to stop...")
	os.Stdin.Read([]byte{0})
	file.Close()
}
