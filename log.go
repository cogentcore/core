package packman

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
)

// Log prints the logs from your app running on the given operating system (android or ios) to the terminal
func Log(osName string, keep bool, allLevel string) error {
	if osName == "ios" {
		return errors.New("ios not supported yet")
	}
	if !keep {
		cmd := exec.Command("adb", "logcat", "-c")
		fmt.Println(cmd.Args)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error clearing logs: %w, %s", err, string(output))
		}
		fmt.Println(string(output))
	}
	cmd := exec.Command("adb", "logcat", "*:"+allLevel, "Go:I", "GoLog:I")
	fmt.Println(cmd.Args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("erroring getting logs: %w", err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting logging: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		t := scanner.Text()
		fmt.Println(t)
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("error logging: %w", err)
	}
	return nil
}
