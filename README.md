# gasms
Grove AppStakes Management System

### Helper Function to get current stakes:
`pkd_mainnet_query application list-application -o json | jq '.applications[]   | select(.delegatee_gateway_addresses[] == "pokt1lf0kekv9zcv9v3wy4v6jx2wh7v4665s8e0sl9s")   | {address, stake_amount: .stake.amount, service_id: .service_configs[].service_id}'`
