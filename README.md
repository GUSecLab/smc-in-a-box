# Shadow smc branch

This is a branch of the dev3.0 branch that was used to run simulations in the Shadow network simulator.

To generate configuration files change the settings in `/local/main.go` run `go build` in the `/local/` folder then run `./local`

Additionally `outputparty/cmd/experiments.json` and `server/cmd/experiments.json` has had their due times modified because Shadow starts its time at 1/1/2000 at 12:00.

The generated configuration files need the hostnames changed because of how the Shadow simulation works. The hostname is server/client + [number] where the number matches the last number of the port number. See the smc-config-files folder in shadow_script for more information. To generate new configuration files see the instructions above but know that you will likely need to change at least some of the hostnames in the files. 

# Simple Prototype System for Private Statistics
A simple prototype system for private statistics,  comprising clients, several secure multiparty computation servers, and output party(e.g., data analyst) written in Golang.

The prototype system simulates a scenario in which the output party designs experiments or conducts surveys to collect statistics, such as a data analyst. In this scenario, the client actively participates in the experiment or survey. Servers play a crucial role in assisting the output party in obtaining statistics without learning the client's input, ensuring the privacy of the client's data remains protected.

## Preparing Configuration and Input 
Each party has a configuration file comprising parameters used in the protocol, alongside an input file. The client's input file (cmd/input.json) has response to the experiment or survey. The server and output party input files (cmd/experiments.json)has experiment details like the experiment due time.

## Running Computation
There are two ways running computation:
1. Seperate execution. This allows each party running on a different machine. 
   
   To run a server with TLS (default), at the folder server/cmd, compile then run
   ```
   ./cmd -confpath="path_to_server_config_file" -inputpath="path_to_experiments_file"
   ```
   Note: use -mode=http to run without TLS

   To run an output party with TLS (default), at the folder outputparty/cmd, compile then run
   ```
   ./cmd -confpath="path_to_output_party_config_file" -inputpath="path_to_experiments_file"
   ```
   Note: use -mode=http to run without TLS

   To run a client, at the folder client/cmd, compile then run
   ```
   ./cmd -confpath=“path_to_client_config_file” -inputpath=“path_to_input_file”
   ``` 
   **Note:** Servers and output party need to start running before clients.
2. One-command local executaion. This allows all parties running on the same machine.
   At folder local, compile then run
   ```
   ./local
   ``` 








  
