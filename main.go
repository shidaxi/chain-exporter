package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

var (
	updates     string
	config      *fetchConfig
	loadSeconds float64
	totalLoaded int64
	eth         *ethclient.Client
)

const erc20AbiDefinition = `[{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"int256"}],"type":"function"}]`

var addr = flag.String("listen-address", ":9060", "The address to listen on for HTTP requests.")
var configPath = flag.String("config", "./config.yaml", "The config file path.")
var chainName = flag.String("chain-name", "eth-goerli", "Chain name.")
var rpcUrl = flag.String("rpc-url", "https://rpc.ankr.com/eth_goerli", "Chain RPC URL.")

type fetchConfig struct {
	ScrapeIntervalSeconds int16             `yaml:"scrapeIntervalSeconds"`
	Balance               map[string]string `yaml:"balance"`
	Erc20balance          map[string]struct {
		ContractAddress string            `yaml:"contractAddress"`
		Accounts        map[string]string `yaml:"accounts"`
	} `yaml:"erc20balance"`
	ContractCall map[string]struct {
		ContractName          string   `yaml:"contractName"`
		ContractAddress       string   `yaml:"contractAddress"`
		ScrapeIntervalSeconds int16    `yaml:"scrapeIntervalSeconds"`
		AbiDefination         string   `yaml:"abiDefination"`
		OutputDecimals        int16    `yaml:"outputDecimals"`
		Args                  []string `yaml:"args"`
	} `yaml:"contractCall"`
	StandardRpcEndpoint     string            `yaml:"standardRpcEndpoint"`
	ReplicaRpcEndpoints     map[string]string `yaml:"replicaRpcEndpoints"`
	HashCheckBackwardOffset uint64            `yaml:"hashCheckBackwardOffset"`
}

type metrics struct {
	balance             *prometheus.GaugeVec
	nounce              *prometheus.GaugeVec
	erc20balance        *prometheus.GaugeVec
	contractData        *prometheus.GaugeVec
	blockHashEigenValue *prometheus.GaugeVec
	stateRootEigenValue *prometheus.GaugeVec
}

func ConnectionToGeth(url string) error {
	var err error
	eth, err = ethclient.Dial(url)
	return err
}

func GetEthBalance(address string) *big.Float {
	balance, err := eth.BalanceAt(context.Background(), common.HexToAddress(address), nil)
	if err != nil {
		fmt.Printf("Error fetching ETH Balance for address: %v\n", address)
	}
	return ToEther(balance)
}

func GetAccountNounce(address string) uint64 {
	nounce, err := eth.NonceAt(context.Background(), common.HexToAddress(address), nil)
	if err != nil {
		fmt.Printf("Error fetching ETH Balance for address: %v\n", address)
	}
	return nounce
}

func CurrentBlock() uint64 {
	block, err := eth.BlockByNumber(context.Background(), nil)
	if err != nil {
		fmt.Printf("Error fetching current block height: %v\n", err)
		return 0
	}
	return block.NumberU64()
}

func ToEther(o *big.Int) *big.Float {
	pul, int := big.NewFloat(0), big.NewFloat(0)
	int.SetInt(o)
	pul.Mul(big.NewFloat(0.000000000000000001), int)
	return pul
}

func CollectBalanceMetric(address string) big.Int {
	return *big.NewInt(0)
}

func CollectErc20BalanceMetric(symbol string, contractAddress string, address string) big.Int {
	return *big.NewInt(0)
}

func CollectContractMetric(address string, calldata []byte) big.Int {
	return *big.NewInt(0)
}

func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		balance: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chain_accountbalance",
			Help: "",
		}, []string{"chainName", "rpcUrl", "accountName", "accountAddress"}),
		nounce: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chain_account_nounce",
			Help: "",
		}, []string{"chainName", "rpcUrl", "accountName", "accountAddress"}),
		erc20balance: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chain_erc20balance",
			Help: "",
		}, []string{"chainName", "rpcUrl", "symbol", "accountName", "accountAddress"}),
		contractData: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chain_contractdata",
			Help: "",
		}, []string{"chainName", "rpcUrl", "contractName", "contractAddress", "methodDef", "args"}),
		blockHashEigenValue: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chain_blockhash_eigenvalue",
			Help: "",
		}, []string{"chainName", "rpcUrl"}),
		stateRootEigenValue: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "chain_stateroot_eigenvalue",
			Help: "",
		}, []string{"chainName", "rpcUrl"}),
	}
	reg.MustRegister(m.balance)
	reg.MustRegister(m.nounce)
	reg.MustRegister(m.erc20balance)
	reg.MustRegister(m.contractData)
	reg.MustRegister(m.blockHashEigenValue)
	reg.MustRegister(m.stateRootEigenValue)
	return m
}

func main() {
	flag.Parse()
	config = config.getConf()
	fmt.Println(config)

	err := ConnectionToGeth(*rpcUrl)
	if err != nil {
		log.Printf(err.Error())
	}

	// Create a new registry.
	reg := prometheus.NewRegistry()

	// Add Go module build info.
	reg.MustRegister(collectors.NewBuildInfoCollector())
	reg.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/.*")}),
	))

	m := NewMetrics(reg)

	// fetch balance
	for k, v := range config.Balance {
		go func(accountName string, accountAddress string) {
			for {
				b, _ := GetEthBalance(accountAddress).Float64()
				n := float64(GetAccountNounce(accountAddress))
				log.Printf("Scrapting balance of account %s: %f\n", accountName, b)
				log.Printf("Scrapting nounce of account %s: %f\n", accountName, n)
				m.balance.WithLabelValues(*chainName, *rpcUrl, accountName, accountAddress).Set(b)
				m.nounce.WithLabelValues(*chainName, *rpcUrl, accountName, accountAddress).Set(n)
				time.Sleep(time.Duration(config.ScrapeIntervalSeconds) * time.Second)
			}
		}(k, v)
	}
	// fetch erc20 balance
	for symbol, erc20Config := range config.Erc20balance {
		abiObj, _ := abi.JSON(strings.NewReader(erc20AbiDefinition))
		erc20Address := common.HexToAddress(erc20Config.ContractAddress)
		for k, v := range erc20Config.Accounts {
			go func(symbol string, erc20Address common.Address, accountName string, accountAddress string) {
				for {
					log.Printf("Scrapting erc20(%s) balance of account %s\n", symbol, accountName)
					callData, err := abiObj.Pack("balanceOf", common.HexToAddress(accountAddress))
					callMsg := ethereum.CallMsg{To: &erc20Address, Data: callData}
					if err != nil {
						panic(err)
					}
					res, err := eth.CallContract(context.Background(), callMsg, nil)
					n := new(big.Int)
					n.SetString(strings.ReplaceAll(hexutil.Encode(res), "0x", ""), 16)
					balance, _ := ToEther(n).Float64()
					m.erc20balance.WithLabelValues(*chainName, *rpcUrl, symbol, accountName, accountAddress).Set(balance)
					time.Sleep(time.Duration(config.ScrapeIntervalSeconds) * time.Second)
				}
			}(symbol, erc20Address, k, v)
		}
	}
	// fetch contract metrics
	for k, v := range config.ContractCall {
		callName := k
		callConfig := v
		abiObj, err := abi.JSON(strings.NewReader(callConfig.AbiDefination))
		if err != nil {
			log.Printf(err.Error())
		}
		var args []interface{}
		var methodName string
		for name, def := range abiObj.Methods {
			methodName = name
			for i, arg := range def.Inputs {
				inputArg := callConfig.Args[i]
				typeString := arg.Type.String()
				if arg.Type.String() == "address" {
					args = append(args, common.HexToAddress(inputArg))
				} else if strings.Contains(typeString, "int") {
					n, _ := strconv.ParseInt(inputArg, 10, 64)
					switch typeString {
					case "int8":
						args = append(args, int8(n))
					case "int16":
						args = append(args, int16(n))
					case "int32":
						args = append(args, int32(n))
					case "int64":
						args = append(args, int64(n))
					case "uint8":
						args = append(args, uint8(n))
					case "uint16":
						args = append(args, uint16(n))
					case "uint32":
						args = append(args, uint32(n))
					case "uint64":
						args = append(args, uint64(n))
					default:
						args = append(args, big.NewInt(n))
					}
				} else if typeString == "bool" {
					r, err := strconv.ParseBool(inputArg)
					if err != nil {
						log.Fatalln(err)
					}
					args = append(args, r)
				} else {
					args = append(args, inputArg)
				}
			}
			break
		}
		callData, err := abiObj.Pack(methodName, args...)
		if err != nil {
			log.Printf(err.Error())
		}
		contractAddress := common.HexToAddress(callConfig.ContractAddress)
		callMsg := ethereum.CallMsg{To: &contractAddress, Data: callData}
		go func() {
			for {
				log.Printf("Scrapting contract data: %s: %s %s\n", callName, callConfig.ContractName, callConfig.ContractAddress)
				res, err := eth.CallContract(context.Background(), callMsg, nil)
				if err != nil {
					panic(err)
				}
				s := hexutil.Encode(res)
				n := new(big.Int)
				n.SetString(s[len(s)-64:], 16)
				f, _ := new(big.Float).SetInt(n.Div(n, math.BigPow(10, int64(callConfig.OutputDecimals)))).Float64()
				m.contractData.WithLabelValues(*chainName, *rpcUrl, callConfig.ContractName, callConfig.ContractAddress, methodName, strings.Join(callConfig.Args, "_")).Set(f)
				time.Sleep(time.Duration(callConfig.ScrapeIntervalSeconds) * time.Second)
			}
		}()
	}

	if config.StandardRpcEndpoint != "" && len(config.ReplicaRpcEndpoints) > 0 {
		go func() {
			for {
				eth, err = ethclient.Dial(config.StandardRpcEndpoint)
				blockNumber, err := eth.BlockNumber(context.Background())
				if err != nil {
					panic(err)
				}
				for _, rpcUrl := range config.ReplicaRpcEndpoints {
					eth, err = ethclient.Dial(rpcUrl)
					blk, err := eth.BlockByNumber(context.Background(), big.NewInt(int64(blockNumber)))
					if err != nil {
						panic(err)
					}
					blockNumberX := blockNumber % 100
					blockHash := blk.Hash().String()
					blockHashX, _ := hexutil.DecodeUint64("0x" + blockHash[len(blockHash)-1:])
					m.blockHashEigenValue.WithLabelValues(*chainName, rpcUrl).Set(float64(blockNumberX + blockHashX))
					stateRootHash := blk.Root().String()
					stateRootHashX, _ := hexutil.DecodeUint64("0x" + stateRootHash[len(stateRootHash)-1:])
					m.stateRootEigenValue.WithLabelValues(*chainName, rpcUrl).Set(float64(blockNumberX + stateRootHashX))
					log.Println(blockNumber, blockHash, stateRootHash)
				}
				time.Sleep(time.Duration(config.ScrapeIntervalSeconds) * time.Second)
			}
		}()
	}

	http.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{},
	))
	block := CurrentBlock()
	fmt.Printf("chain-exporter has started on port %v using Geth server: %v at block #%v\n", *addr, *rpcUrl, block)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func (c *fetchConfig) getConf() *fetchConfig {

	yamlFile, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}
