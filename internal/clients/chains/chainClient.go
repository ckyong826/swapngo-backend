package chains

type IChainClient interface {
	GenerateAddress() (address string, privateKey string, err error)
}
