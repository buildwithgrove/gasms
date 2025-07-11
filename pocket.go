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
}

func QueryApplications(rpcEndpoint, gateway string) ([]Application, error) {
	// Build the command equivalent to:
	// pocketd q application list-application -o json $MAINNODE | jq '.applications[] | select(.delegatee_gateway_addresses[] == "gateway") | {address, stake_amount: .stake.amount, service_id: .service_configs[].service_id}'

	cmd := exec.Command("pocketd", "q", "application", "list-application", "-o", "json", "--node", rpcEndpoint)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute pocketd command: %w", err)
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

		applications = append(applications, Application{
			Address:     app.Address,
			StakeAmount: app.Stake.Amount,
			ServiceID:   serviceID,
			StakePOKT:   stakePOKT,
		})
	}

	return applications, nil
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
