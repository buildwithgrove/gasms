package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type Application struct {
	Address     string  `json:"address"`
	StakeAmount string  `json:"stake_amount"`
	ServiceID   string  `json:"service_id"`
	StakePOKT   float64 // Calculated field for display
	BalancePOKT float64 // Bank balance in POKT
}

func QueryApplications(rpcEndpoint, gateway, keyringBackend, pocketdHome, networkName string) ([]Application, error) {
	// Build the command equivalent to:
	// pocketd q application list-application -o json $MAINNODE | jq '.applications[] | select(.delegatee_gateway_addresses[] == "gateway") | {address, stake_amount: .stake.amount, service_id: .service_configs[].service_id}'
	// Use --limit 10000 to ensure we get all applications (pagination workaround)

	// Determine chain ID based on network name
	var chainID string
	switch networkName {
	case "pocket":
		chainID = "pocket"
	case "pocket-beta":
		chainID = "pocket-beta"
	default:
		return nil, fmt.Errorf("unsupported network: %s", networkName)
	}

	args := []string{"q", "application", "list-application", "-o", "json", "--node", rpcEndpoint, "--chain-id", chainID, "--limit", "10000"}
	// Only add --home flag for query commands (keyring-backend not needed for queries)
	if pocketdHome != "" {
		args = append(args, "--home="+pocketdHome)
	}
	cmd := exec.Command("pocketd", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute pocketd command: %w, output: %s", err, string(output))
	}

	// Parse the JSON output
	var response struct {
		Applications []struct {
			Address string `json:"address"`
			Stake   struct {
				Amount string `json:"amount"`
			} `json:"stake"`
			ServiceConfigs []struct {
				ServiceID string `json:"service_id"`
			} `json:"service_configs"`
			DelegateeGatewayAddresses []string `json:"delegatee_gateway_addresses"`
		} `json:"applications"`
	}

	err = json.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	var applications []Application

	for _, app := range response.Applications {
		// Check if this app has our gateway
		hasGateway := false
		for _, gw := range app.DelegateeGatewayAddresses {
			if gw == gateway {
				hasGateway = true
				break
			}
		}

		if !hasGateway {
			continue
		}

		// Get service ID (use first one if multiple)
		serviceID := "-"
		if len(app.ServiceConfigs) > 0 {
			serviceID = app.ServiceConfigs[0].ServiceID
		}

		// Convert stake amount to POKT (divide by 1,000,000)
		stakeAmount, err := strconv.ParseFloat(app.Stake.Amount, 64)
		if err != nil {
			stakeAmount = 0
		}
		stakePOKT := stakeAmount / 1_000_000

		// Query bank balance for this application
		balancePOKT, err := QueryBankBalance(app.Address, rpcEndpoint, keyringBackend, pocketdHome)
		if err != nil {
			// If balance query fails, set to 0 and continue
			balancePOKT = 0
		}

		applications = append(applications, Application{
			Address:     app.Address,
			StakeAmount: app.Stake.Amount,
			ServiceID:   serviceID,
			StakePOKT:   stakePOKT,
			BalancePOKT: balancePOKT,
		})
	}

	return applications, nil
}

func QueryBankBalance(address, rpcEndpoint, keyringBackend, pocketdHome string) (float64, error) {
	args := []string{"q", "bank", "balances", address, "--node", rpcEndpoint, "--output", "json"}
	// Only add --home flag for query commands (keyring-backend not needed for queries)
	if pocketdHome != "" {
		args = append(args, "--home="+pocketdHome)
	}
	cmd := exec.Command("pocketd", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to execute pocketd balance query: %w, output: %s", err, string(output))
	}

	// Parse the JSON output
	var response struct {
		Balances []struct {
			Amount string `json:"amount"`
			Denom  string `json:"denom"`
		} `json:"balances"`
	}

	err = json.Unmarshal(output, &response)
	if err != nil {
		return 0, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Find upokt balance
	for _, balance := range response.Balances {
		if balance.Denom == "upokt" {
			amount, err := strconv.ParseFloat(balance.Amount, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse balance amount: %w", err)
			}
			// Convert from upokt to POKT (divide by 1,000,000)
			return amount / 1_000_000, nil
		}
	}

	// No upokt balance found
	return 0, nil
}

func TruncateAddress(address string, maxLen int) string {
	if len(address) <= maxLen {
		return address
	}
	if maxLen < 10 {
		return address[:maxLen]
	}
	// Show first 6 and last 4 characters with ... in between
	return address[:6] + "..." + address[len(address)-4:]
}

// AppendPocketdFlags adds optional keyring-backend and home flags to pocketd command args
func AppendPocketdFlags(args []string, keyringBackend, pocketdHome string) []string {
	if keyringBackend != "" {
		args = append(args, "--keyring-backend="+keyringBackend)
	}
	if pocketdHome != "" {
		args = append(args, "--home="+pocketdHome)
	}
	return args
}
