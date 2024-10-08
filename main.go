package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Args struct {
	Protocol          string `json:"Protocol"`
	GetServerData     bool   `json:"GetServerData"`
	TestRunimeSeconds int    `json:"TestRunimeSeconds"`
	ReportIntervall   int    `json:"ReportIntervall"`
	Bandwidth         string `json:"Bandwidth"`
	ParallelStreams   int    `json:"ParallelStreams"`
	JSONformat        bool   `json:"JSONformat"`
}

type Config struct {
	Connections    int      `json:"Connections"`
	Names          []string `json:"Names"`
	ServerIPList   []string `json:"ServerIPList"`
	ServerPortList []int    `json:"ServerPortList"`
	Args           Args     `json:"Args"`
}

func RunTest(config *Config, reverseMode bool, done chan bool) {

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

	outputDir := "./" + timestamp // Define the output directory

	var wg sync.WaitGroup
	errChan := make(chan error, len(config.ServerIPList)) // Buffered channel to collect errors

	for i := range config.ServerIPList {

		outputFileName := config.Names[i]
		wg.Add(1)

		go RunIperf(config, i, reverseMode, outputFileName, outputDir, &wg, errChan)
	}

	wg.Wait()      // Wait for all goroutines to finish
	close(errChan) // Close the channel to stop range loop

	for err := range errChan {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	done <- true
}

func GenerateIperfArgs(config *Config, siteNum int, reverseMode bool) []string {
	args := []string{
		"-c", config.ServerIPList[siteNum],
		"-p", strconv.Itoa(config.ServerPortList[siteNum]),
		"-t", strconv.Itoa(config.Args.TestRunimeSeconds),
		"-i", strconv.Itoa(config.Args.ReportIntervall),
	}

	// Add the -u flag if the protocol is UDP
	if config.Args.Protocol == "UDP" {
		args = append([]string{"-u"}, args...)
	}

	// Add the -R flag if reverse mode is true
	if reverseMode {
		args = append([]string{"-R"}, args...)
	}

	// Add the --get-server-output flag if we want the server output
	if config.Args.GetServerData {
		args = append([]string{"--get-server-output"}, args...)
	}

	if config.Args.Bandwidth != "" {
		args = append([]string{"-bw " + config.Args.Bandwidth}, args...)
	}

	if config.Args.ParallelStreams > 0 {
		args = append([]string{"-P " + strconv.Itoa(config.Args.ParallelStreams)}, args...)
	}

	if config.Args.JSONformat {
		args = append([]string{"-J"}, args...)
	}

	return args
}

func RunIperf(config *Config, siteNum int, reverseMode bool, outputFileName string, outputDir string, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done() // Decrement the counter when the goroutine completes

	fmt.Println("Running iperf for", outputFileName)

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Args.TestRunimeSeconds+5)*time.Second)
	defer cancel()

	args := GenerateIperfArgs(config, siteNum, reverseMode)

	// Create the iperf command with context
	cmd := exec.CommandContext(ctx, "iperf3", args...)

	// Run the iperf command and capture the output
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		errChan <- fmt.Errorf("\033[31miperf command timed out: %v\033[0m", outputFileName) // Red
		return
	}

	if err != nil {
		errChan <- fmt.Errorf("\033[31merror running iperf: %v on site %v\033[0m", err, outputFileName) // Red
		return
	}

	// Open the file for writing the output
	file, err := os.Create(outputDir + "/" + outputFileName + ".txt")
	if err != nil {
		errChan <- fmt.Errorf("\033[31merror creating output file: %v\033[0m", err) // Red
		return
	}
	defer file.Close()

	// Write the output to the file
	_, err = file.WriteString(string(output))
	if err != nil {
		errChan <- fmt.Errorf("\033[31merror writing output to file: %v\033[0m", err) // Red
		return
	}

	time.Sleep(2 * time.Second) // Simulate some work being done

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
		fmt.Printf("Something went wrong during the validation of the config\n")
		return
	}

	reverseMode := false
	quickRun := false

	// Iterate over os.Args to check for 'r' and 'q'
	for _, arg := range os.Args[1:] {
		if arg == "r" {
			reverseMode = true
		}
		if arg == "q" {
			quickRun = true
		}
	}

	// Handle quick run mode ('q')
	if quickRun {
		done := make(chan bool)
		go ShowSpinnerAnimation(done)
		RunTest(config, reverseMode, done)
		return
	}

	input := ""

	// menu loop
	for {
		PrintTitle()
		PrintMenu()
		fmt.Printf("\n")
		fmt.Printf(">")

		_, err := fmt.Scan(&input)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// visualize config
		if input == "1" {
			input = ""
			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			cmd.Run()
			PrintConfig(config, reverseMode)
			for {
				_, err := fmt.Scan(&input)
				if err != nil {
					fmt.Println("Error:", err)
				}

				if input == "x" {
					cmd := exec.Command("clear")
					cmd.Stdout = os.Stdout
					cmd.Run()
					break
				}
			}
		}

		// run tests
		if input == "2" {
			DeleteUpperLines(8)

			done := make(chan bool)
			go ShowSpinnerAnimation(done)
			RunTest(config, reverseMode, done)

			return
		}

		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}

}

func PrintConfig(config *Config, reverseMode bool) {

	fmt.Println("Config data - " + "\033[35m" + "Type in 'x'" + "\033[0m" + " to return to the main menu")
	fmt.Printf("\n")
	fmt.Printf("\033[33mIperf Arguments\033[0m\n\n")
	fmt.Printf("Example iperf command to be run (IP and Port change according to data provided)\n")
	fmt.Printf("\033[32miperf3 %s\033[0m\n", strings.Join(GenerateIperfArgs(config, 0, reverseMode), " "))

	fmt.Printf("\n")

	fmt.Printf("\033[33mServerlist to connect to\033[0m\n\n")
	for i, ip := range config.ServerIPList {
		fmt.Printf("Serversite %d: \n", i)
		fmt.Printf("- IP: \033[34m%s\033[0m \n", ip)                         // IP in blue
		fmt.Printf("- Port: \033[34m%d\033[0m \n", config.ServerPortList[i]) // Port in blue
		fmt.Printf("\n")
	}

	fmt.Printf("\n")
	fmt.Printf(">")
}

func PrintMenu() {
	// ANSI escape code for the desired color (#aa9340)
	color := "\033[38;2;170;147;64m"
	reset := "\033[0m" // Reset color to default

	// Menu with colored lines
	title := color + `
╔═════════════════════════════════════════════╗
║` + reset + `                 Main Menu                   ` + color + `║
╠═════════════════════════════════════════════╣
║` + reset + `  1. Check the provided config               ` + color + `║
║` + reset + `  2. Run all tests                           ` + color + `║
╚═════════════════════════════════════════════╝` + reset

	// Print the menu
	fmt.Println(title)
}

func DeleteUpperLines(n int) {
	for i := 0; i < n; i++ {
		// Move the cursor up one line and clear the line
		fmt.Printf("\033[1A\033[K")
	}
}

func ShowSpinnerAnimation(done chan bool) {
	time.Sleep(200 * time.Millisecond)

	fmt.Printf("\n")
	// Spinner frames for animation
	frames := []string{"⠋", "⠙", "⠸", "⠴", "⠦", "⠧", "⠇", "⠏"}

	// Animation loop
	for {
		for _, frame := range frames {
			select {
			case <-done:
				fmt.Printf("\r%s \033[32mDone!\033[0m                    \n", frame)
				return
			default:
				fmt.Printf("\r%s \033[33mWaiting...\033[0m", frame) // Print the spinner first, then "Waiting..."
				time.Sleep(100 * time.Millisecond)                  // Delay between frames
			}
		}
	}
}

func PrintTitle() {
	// Title 1 with dark blue color (#00345f)
	fmt.Printf("\n")
	title_1 := "\033[38;2;0;52;95m" + `▗▖ ▗▖▗▄▄▄ ▗▄▄▖ 
▐▌ ▐▌▐▌  █▐▌ ▐▌
▐▌ ▐▌▐▌  █▐▛▀▚▖
▐▙█▟▌▐▙▄▄▀▐▌ ▐▌` + "\033[0m\n"

	// Title 2 with the desired color (#aa9340)
	title_2 :=

		`
▗▖  ▗▖▗▄▄▄▖▗▄▄▄▖▗▖  ▗▖▗▄▄▄▖ ▗▄▄▖ ▗▄▄▖
▐▛▚▖▐▌▐▌     █  ▐▛▚▞▜▌▐▌   ▐▌   ▐▌   
▐▌ ▝▜▌▐▛▀▀▘  █  ▐▌  ▐▌▐▛▀▀▘ ▝▀▚▖ ▝▀▚▖
▐▌  ▐▌▐▙▄▄▖  █  ▐▌  ▐▌▐▙▄▄▖▗▄▄▞▘▗▄▄▞▘`

	fmt.Printf(title_1)
	fmt.Println(title_2)
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
