package contractapi

type myContract struct {
	Contract
	called []string
}

func (mc *myContract) ReturnsString() string {
	return "Some string"
}

type customContext struct {
	TransactionContext
	prop1 string
}
