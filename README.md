                                                 =====                                    
                                              ==========                                  
                                             ===========                                  
                                            ============                                  
                                =====       ============                                  
                               =========    ============                                  
                              =============  ==========                                   
                              ======================                                      
                              ================                                            
                              ================          ========                          
                               ===============      =============                         
                                  ============   ================                         
                      =====          ============================                         
                    =========            === ====================                         
                    =============           ====================                          
                    ================        ==================                            
                    ===================     ===============                               
                    =======================  ===========                                  
                     ================================          =======                    
                       =======================             ============                   
                           ===================           ==============                   
                              ================        =================                   
                                 =============    =====================                   
                                    ========== ========================                   
                                       ===============================                    
                                            ======================                        
                                            ===================                           
                                            ================                              
                                            ============                                  
                                     ================                                     
                                     ========= ====                                       
                                     =========                                            
                                     =========                                            
                                     =========                                            
                                     =========                                            
                                      =======    
		   ___                     _             ___ _        _                         
		  / __|_ _ _____ _____    /_\  _ __ _ __/ __| |_ __ _| |_____ ___               
		 | (_ | '_/ _ \ V / -_)  / _ \| '_ \ '_ \__ \  _/ _` | / / -_|_-<               
		  \___|_| \___/\_/\___| /_/ \_\ .__/ .__/___/\__\__,_|_\_\___/__/_              
		 |  \/  |__ _ _ _  __ _ __ _ _|_|_ |_| ___ _ _| |_  / __|_  _ __| |_ ___ _ __   
		 | |\/| / _` | ' \/ _` / _` / -_) '  \/ -_) ' \  _| \__ \ || (_-<  _/ -_) '  \  
		 |_|  |_\__,_|_||_\__,_\__, \___|_|_|_\___|_||_\__| |___/\_, /__/\__\___|_|_|_| 
		             ___   _   |___/_  __ ___                    |__/                   
		  ___ _ _   / __| /_\ / __|  \/  / __|                                          
		 / _ \ '_| | (_ |/ _ \\__ \ |\/| \__ \                                          
		 \___/_|    \___/_/ \_\___/_|  |_|___/
# `gasms`

## What is `gasms`?
The **G**rove **A**ppStakes **M**anagement **S**ystem or **GASMS** is a **TUI** (**T**erminal **U**ser **I**nterface) tool designed to help Gateways on [Pocket Network](https://pocket.network) manage the staking balances of applications required to send traffic. 

## Features
- **Real-time Application Monitoring**: Track stakes, service IDs, and gateway assignments
- **`vi`-style Keybindings**: Familiar navigation for terminal power users
- **Multi-network Support**: Configure multiple networks (beta, main, etc.)
- **Fast & Lightweight**: Single binary with no dependencies
- **Search & Filter**: Find applications quickly with / search
- **Automatic Refresh**: Keep data current with `r` refresh

## Installation
### From Source
```bash
# Clone the repository
git clone <your-repo-url>
cd gasms

# Build the binary
make build

# Install to your PATH
make install
```

### Cross-Platform Builds
```bash
# Build for all platforms
make build-all

# Binaries will be in bin/ directory:
# - gasms-linux-amd64
# - gasms-linux-arm64  
# - gasms-darwin-amd64
# - gasms-darwin-arm64
# - gasms-windows-amd64.exe
```

### Configuration
Create a `config.yaml` file in your working directory:
```yaml
config:
  networks: 
    beta:
      rpc_endpoint: <NETWORK_RPC_URL>
      gateways: 
        - <GATEWAY_ADDRESS>
      applications:
        - application1
        # ... more applications
    main:
      rpc_endpoint: <NETWORK_RPC_URL>
      gateways: 
        - <GATEWAY_ADDRESS>
      applications:
        - application1
        # ... more applications
```

## Usage
```bash
# Run GASMS
./gasms

# Or if installed to PATH
gasms
```

### Keybindings
| Key | Action |
|-----|--------|
| `q` | Quit application |
| `r` | Refresh data |
| `/` | Search applications |
| `n` | Browse and Change Networks |
| `:` | Enter command mode |
| `↑/k` | Move cursor up |
| `↓/j` | Move cursor down |
| `g` | Go to top |
| `G` | Go to bottom |
| `Esc` | Cancel command/search |

### Commands
In command mode (press :):

`:q` or `:quit` - Quit application
`:n` or `:network` - Browse and Change Networks (i.e. Main, Beta, etc.)

## Development
### Prerequisites
- [`Go 1.24+`](https://go.dev/doc/install)
- [`Make`](https://www.gnu.org/software/make/)
- [`pocketd`](https://github.com/pokt-network/poktroll)

### Development Setup
```bash
# Install dependencies
make deps

# Run in development mode
make run

# Run with live reload (requires air)
make dev

# Format code
make fmt

# Run tests
make test
```
### Dependencies
- [`bubbletea`](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [`lipgloss`](https://github.com/charmbracelet/lipgloss) - Styling and layout
- `yaml.v3` - YAML configuration parsing

## Inspiration
This tool is heavily inspired by 🐶[`k9s`](https://github.com/derailed/k9s) a TUI for managing Kubernetes clusters.

## Helper Functions
### Helper Function to get current stakes:
`pkd_mainnet_query application list-application -o json | jq '.applications[]   | select(.delegatee_gateway_addresses[] == "pokt1lf0kekv9zcv9v3wy4v6jx2wh7v4665s8e0sl9s")   | {address, stake_amount: .stake.amount, service_id: .service_configs[].service_id}'`
