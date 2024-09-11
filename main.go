package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"
)

type Config struct {
	NumServers     int
	ServerIPList   []string
	ServerPortList []int
}

func RunIperf(config *Config, outputFileName string, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done() // Decrement the counter when the goroutine completes

	/* Uncomment and replace with actual iperf execution code
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Windows specific command - Not implemented yet
		// cmd = exec.Command("iperf.exe", "-c", server, "-t", fmt.Sprintf("%d", duration))
		fmt.Printf("Windows as a runtime enviroment is not yet implemented")
		errChan <- fmt.Errorf("Windows environment not supported")
		return
	} else {
		// Linux or other Unix-like OS command
		// cmd = exec.Command("iperf", "-c", server, "-t", fmt.Sprintf("%d", duration))
	}

	// Run the iperf command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		errChan <- fmt.Errorf("Error running iperf: %v", err)
		return
	}

	// Open the file for writing the output
	file, err := os.Create(outputFileName + ".txt")
	if err != nil {
		errChan <- fmt.Errorf("Error creating output file: %v", err)
		return
	}
	defer file.Close()

	// Write the output to the file
	_, err = file.WriteString(string(output))
	if err != nil {
		errChan <- fmt.Errorf("Error writing output to file: %v", err)
		return
	}
	*/

	fmt.Println("Running iperf for", outputFileName)
	time.Sleep(2 * time.Second) // Simulate some work being done

	// Simulate an error for demonstration purposes
	// Remove or handle properly in real code
	if outputFileName == "Site_1_iperf_output" { // Simulate an error for specific case
		errChan <- fmt.Errorf("Simulated error for %s", outputFileName)
		return
	}

	errChan <- nil // Indicate success
}

func ParseConfig() *Config {
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return nil
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return nil
	}

	var config Config
	if err := json.Unmarshal(bytes, &config); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return nil
	}

	return &config
}

func main() {
	config := ParseConfig()

	if config == nil {
		fmt.Printf("Something went wrong during the validation of the config")
		return
	}

	timestamp := time.Now().Format("Mon Jan 2 15:04:05 MST 2006")

	err := os.Mkdir("./"+timestamp, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	err = CopyFile("config.json", "./"+timestamp+"/config.json")
	if err != nil {
		fmt.Printf("Error copying config.json into new directory: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(config.ServerIPList)) // Buffered channel to collect errors

	for i := range config.ServerIPList {

		outputFileName := "Site_" + strconv.Itoa(i) + "_iperf_output"

		wg.Add(1)

		go RunIperf(config, outputFileName, &wg, errChan)
	}

	wg.Wait()      // Wait for all goroutines to finish
	close(errChan) // Close the channel to stop range loop

	for err := range errChan {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	// TODO inform about error
	fmt.Println("All Iperf tests completed.")
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}
