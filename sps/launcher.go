package sps

import (
	"fmt"
	"os/exec"
)

const binName = "shairport-sync"

func SpawnProcess() {
	dateCmd := exec.Command(binName, "-V")
	// cmdIn, err := dateCmd.StdinPipe()
	// if err != nil {
	// 	panic(err)
	// }
	// cmdOut, err := dateCmd.StdoutPipe()
	// if err != nil {
	// 	panic(err)
	// }
	// cmdErr, err := dateCmd.StderrPipe()
	// if err != nil {
	// 	panic(err)
	// }
	dateOut, err := dateCmd.Output()
	if err != nil {
		panic(err)
	}
	fmt.Println("> date")
	fmt.Println(string(dateOut))

	_, err = exec.Command("date", "-x").Output()
	if err != nil {
		switch e := err.(type) {
		case *exec.Error:
			fmt.Println("failed executing:", err)
		case *exec.ExitError:
			fmt.Println("command exit rc =", e.ExitCode())
		default:
			panic(err)
		}
	}
}
