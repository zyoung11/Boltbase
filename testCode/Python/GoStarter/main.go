package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	runningProcesses = make(map[string]*exec.Cmd)
	mutex            = &sync.Mutex{}
)

func ExecuteFile(filePath string) error {
	cmd := exec.Command(filePath)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start file: %w", err)
	}

	mutex.Lock()
	runningProcesses[filePath] = cmd
	mutex.Unlock()

	go func() {
		cmd.Wait()
		mutex.Lock()
		delete(runningProcesses, filePath)
		mutex.Unlock()
	}()

	fmt.Printf("Started process for %s with PID: %d\n", filePath, cmd.Process.Pid)
	return nil
}

func StopProcess(path string) error {
	mutex.Lock()
	defer mutex.Unlock()
	cmd, ok := runningProcesses[path]
	if !ok {
		fmt.Printf("Process not found for path: %s. It might have already terminated.\n", path)
		return nil
	}
	err := cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}
	delete(runningProcesses, path)
	fmt.Printf("Killed process for %s\n", path)
	return nil
}

func ExecuteCommandSync(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %s: %w", command, err)
	}
	return nil
}

func main() {
	if err := ExecuteFile("./Boltbase"); err != nil {
		fmt.Printf("Failed to start Boltbase: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Waiting for Boltbase to start...")
	time.Sleep(2 * time.Second)

	fmt.Println("Running python script...")
	if err := ExecuteCommandSync("uv run main.py"); err != nil {
		fmt.Printf("Python script failed: %v\n", err)
	}

	fmt.Println("Stopping Boltbase...")
	if err := StopProcess("./Boltbase"); err != nil {
		fmt.Printf("Failed to stop Boltbase process: %v\n", err)
	}

	fmt.Println("Removing Boltbase.db...")
	if err := os.Remove("Boltbase.db"); err != nil {
		fmt.Printf("Failed to remove Boltbase.db: %v\n", err)
	}

	fmt.Println("All done.")
}
