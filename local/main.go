package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	client_gen "example.com/SMC/client/scripts/generator"
	output_gen "example.com/SMC/outputparty/scripts/generator"
	server_gen "example.com/SMC/server/scripts/generator"
)

func main() {

	n_client := 50
	n_exp := 1
	n_input := []int{2} // For the first experiment, client's input has one value

	n_server := 6
	server_port := []string{"50001", "50002", "50003", "50004", "50005", "50006"} //port for each server

	n_outputparty := 1
	op_port := []string{"60000"}

	clientShareDue := "2024-04-05 17:46:00"
	t1 := 2 // ComplaintDue = ClientShareDue + t1
	t2 := 5 // MaskedShareDue = ClientShareDue + t2
	t3 := 8 // ServerShareDue = ClientShareDue + t3

	client_gen.GenerateClientConfig(n_client, "client_template.json", "./client_config")

	client_gen.GenerateClientInput(n_client, n_exp, n_input, "./client_input")

	server_gen.GenerateServerConfigLocal(n_server, server_port, "server_template.json", "./server_config/")

	server_gen.GenerateServerInput(n_exp, clientShareDue, t1, t2, "http://127.0.0.1:60000/serverShare/", "./server_input")

	output_gen.GenerateOPConfig(n_outputparty, op_port, "outputparty_template.json", "./op_config")

	output_gen.GenerateOPInput(n_exp, clientShareDue, t3, "./op_input")

	// run(n_server, n_outputparty, n_client)

}

func run(n_server, n_outputparty, n_client int) {

	l1 := n_server + n_outputparty
	firstGroup := make([][]string, l1)
	for i := 0; i < n_server; i++ {
		firstGroup[i] = make([]string, 4)
		firstGroup[i][0] = "../server/cmd/cmd"
		firstGroup[i][1] = fmt.Sprintf("-confpath=./server_config/config_s%s.json", strconv.Itoa(i+1))
		firstGroup[i][2] = "-inputpath=./server_input/experiments.json"
		firstGroup[i][3] = "-mode=http"
	}

	for i := n_server; i < l1; i++ {
		firstGroup[i] = make([]string, 4)
		firstGroup[i][0] = "../outputparty/cmd/cmd"
		firstGroup[i][1] = fmt.Sprintf("-confpath=./op_config/config_op%s.json", strconv.Itoa(i-n_server+1))
		firstGroup[i][2] = "-inputpath=./op_input/experiments.json"
		firstGroup[i][3] = "-mode=http"
	}

	secondGroup := make([][]string, n_client)
	for i := 0; i < n_client; i++ {
		secondGroup[i] = make([]string, 3)
		secondGroup[i][0] = "../client/cmd/cmd"
		secondGroup[i][1] = fmt.Sprintf("-confpath=./client_config/config_c%s.json", strconv.Itoa(i+1))
		secondGroup[i][2] = fmt.Sprintf("-inputpath=./client_input/input_c%s.json", strconv.Itoa(i+1))
	}

	var wg sync.WaitGroup

	// Execute the first group of processes in parallel
	for _, cmd := range firstGroup {
		wg.Add(1)
		go executeFirstGroup(cmd[0], cmd[1], cmd[2], cmd[3], &wg)
	}

	time.Sleep(30 * time.Second)

	for _, cmd := range secondGroup {
		wg.Add(1)
		go executeSecondGroup(cmd[0], cmd[1], cmd[2], &wg)
	}

	// Wait for all commands to finish
	wg.Wait()
}

func executeFirstGroup(command, conf_path, input_path, mode string, wg *sync.WaitGroup) {
	defer wg.Done()

	cmd := exec.Command(command, conf_path, input_path, mode)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error executing command %s with %s: %v\n", command, conf_path, err)
		return
	}
}

func executeSecondGroup(command string, conf_path string, input_path string, wg *sync.WaitGroup) {
	defer wg.Done()

	cmd := exec.Command(command, conf_path, input_path)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error executing command %s with %s: %v\n", command, conf_path, err)
		return
	}
}
