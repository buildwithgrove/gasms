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
- **Application Management**: Upstake, fund, and view detailed information for applications
- **`vi`-style Keybindings**: Familiar navigation for terminal power users
- **Multi-network Support**: Configure multiple networks (beta, main, etc.)
- **Fast & Lightweight**: Single binary with no dependencies
- **Search & Filter**: Find applications quickly with / search
- **Automatic Refresh**: Keep data current with `r` refresh
- **Transaction Tracking**: View transaction hashes for upstake and fund operations

## Video Guide
[![GASMS Demo](https://img.youtube.com/vi/p_h-Ui6uls8/0.jpg)](https://www.youtube.com/watch?v=p_h-Ui6uls8)

*Click to watch a demonstration of GASMS features and functionality*

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
  # Stake threshold configuration (denominated in uPOKT)
  thresholds:
    warning_threshold: 2000000000  # 2000 POKT in uPOKT
    danger_threshold: 1000000000   # 1000 POKT in uPOKT
  
  networks: 
    pocket:
      rpc_endpoint: <NETWORK_RPC_URL>
      gateways: 
        - <GATEWAY_ADDRESS>
      bank: <BANK_ADDRESS>  # Required for upstake and fund operations
      applications:
        - application1
        # ... more applications
    pocket-beta:
      rpc_endpoint: <NETWORK_RPC_URL>
      gateways: 
        - <GATEWAY_ADDRESS>
      bank: <BANK_ADDRESS>  # Required for upstake and fund operations
      applications:
        - application1
        # ... more applications
```

## Usage
```bash
# Run from repo
make run

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
| `u` | Upstake selected application |
| `f` | Fund selected application |
| `Enter` | Show application details |
| `↑/k` | Move cursor up |
| `↓/j` | Move cursor down |
| `g` | Go to top |
| `G` | Go to bottom |
| `Esc` | Cancel command/search or return to table view |

### Commands
In command mode (press :):

#### General Commands
`:q` or `:quit` - Quit application
`:n` or `:network` - Browse and Change Networks (i.e. pocket, pocket-beta, etc.)
`:show` - Show detailed information for selected application

#### Application Management
`:u <amount>` or `:upstake <amount>` - Increase stake of selected application by amount (in POKT)
  - Example: `:u 1000` adds 1000 POKT to current stake
  - Displays transaction hash for 10 seconds after completion
  
`:f <amount>` or `:fund <amount>` - Send tokens to selected application (in POKT)
  - Example: `:f 500` sends 500 POKT to the application
  - Displays transaction hash for 10 seconds after completion

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

## Scripts

The `scripts/` directory contains utility scripts for common Pocket Network operations:

### `stake-applications`
Automates the process of staking or restaking applications on the Pocket Network.

**Features:**
- Incrementally increases stake amounts for existing applications
- Handles new application staking with initial amounts
- Supports single application or batch processing
- Dry-run mode for testing commands before execution
- Multi-network support (mainnet/beta)

**Usage:**
```bash
./scripts/stake-applications --network main --home ~/.poktroll --restake-amount 5000000000 -f app-svc-map.txt
```

**Input File Format:** Tab-separated file with application addresses and service IDs:
```
pokt165uqhq43j7xqjnlv53kzsfjn55kecpg4ejyfns	ethereum-mainnet
pokt1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0	polygon-mainnet
```

### `fund-accounts`
Generates multi-send commands to fund multiple accounts in a single transaction.

**Features:**
- Creates efficient batch funding transactions
- Atomic operation (all transfers succeed or fail together)
- Calculates total amounts and fees
- Supports both mainnet and beta networks
- Saves generated commands for easy execution

**Usage:**
```bash
./scripts/fund-accounts sender_key addresses.txt 5005000000upokt main ~/.poktroll/
```

**Input File Format:** Simple text file with one address per line:
```
pokt165uqhq43j7xqjnlv53kzsfjn55kecpg4ejyfns
pokt1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0
pokt198765432109876543210987654321098765432
```

Both scripts include comprehensive help documentation accessible with the `-h` or `--help` flag.

## Helper Functions
### Helper Function to get current stakes:
`pkd_mainnet_query application list-application -o json | jq '.applications[]   | select(.delegatee_gateway_addresses[] == "pokt1lf0kekv9zcv9v3wy4v6jx2wh7v4665s8e0sl9s")   | {address, stake_amount: .stake.amount, service_id: .service_configs[].service_id}'`
