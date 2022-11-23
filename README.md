# Chain Exporter

1. get balance
1. get erc20 balance
1. get contract data from readonly function


# Build
```
make build
```

# Run
```
./chain-exporter -config config.example.yaml -chain-name eth-mainnet -rpc-url https://rpc.ankr.com/eth
```

# Get Metrics

```
curl localhost:9060/metrics -s |grep chain_ | grep -v '#'

chain_accountbalance{accountAddress="0x70997970C51812dc3A010C7d01b50e0d17dc79C8",accountName="hardhat1",chainName="eth-mainnet",rpcUrl="https://rpc.ankr.com/eth"} 0
chain_accountbalance{accountAddress="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",accountName="hardhat0",chainName="eth-mainnet",rpcUrl="https://rpc.ankr.com/eth"} 3.7768296076782e-05
chain_contractdata{args="",chainName="eth-mainnet",contractAddress="0xdac17f958d2ee523a2206206994597c13d831ec7",contractName="UsdtERC20Token",methodDef="totalSupply",rpcUrl="https://rpc.ankr.com/eth"} 3.2297366521e+10
chain_contractdata{args="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",chainName="eth-mainnet",contractAddress="0xdac17f958d2ee523a2206206994597c13d831ec7",contractName="UsdtERC20Token",methodDef="balanceOf",rpcUrl="https://rpc.ankr.com/eth"} 0
chain_erc20balance{accountAddress="0x70997970C51812dc3A010C7d01b50e0d17dc79C8",accountName="hardhat1",chainName="eth-mainnet",rpcUrl="https://rpc.ankr.com/eth",symbol="shib"} 0
chain_erc20balance{accountAddress="0x70997970C51812dc3A010C7d01b50e0d17dc79C8",accountName="hardhat1",chainName="eth-mainnet",rpcUrl="https://rpc.ankr.com/eth",symbol="usdt"} 0
chain_erc20balance{accountAddress="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",accountName="hardhat0",chainName="eth-mainnet",rpcUrl="https://rpc.ankr.com/eth",symbol="shib"} 0
chain_erc20balance{accountAddress="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",accountName="hardhat0",chainName="eth-mainnet",rpcUrl="https://rpc.ankr.com/eth",symbol="usdt"} 0
```