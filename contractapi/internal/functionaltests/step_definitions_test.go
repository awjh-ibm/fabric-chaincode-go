package functionaltests

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	"github.com/DATA-DOG/godog/gherkin"
	"github.com/hyperledger/fabric-chaincode-go/contractapi"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/functionaltests/contracts/complexcontract"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/functionaltests/contracts/extendedsimplecontract"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/functionaltests/contracts/simplecontract"
	"github.com/hyperledger/fabric-chaincode-go/contractapi/internal/functionaltests/contracts/utils"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-protos-go/peer"
)

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

var contractsMap map[string]contractapi.ContractInterface = map[string]contractapi.ContractInterface{
	"SimpleContract":         new(simplecontract.SimpleContract),
	"ExtendedSimpleContract": NewExtendedContract(),
	"ComplexContract":        NewComplexContract(),
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)
}

func NewExtendedContract() *extendedsimplecontract.ExtendedSimpleContract {
	esc := new(extendedsimplecontract.ExtendedSimpleContract)

	esc.SetTransactionContextHandler(new(utils.CustomTransactionContext))
	esc.SetBeforeTransaction(utils.GetWorldState)
	esc.SetUnknownTransaction(utils.UnknownTransactionHandler)

	return esc
}

func NewComplexContract() *complexcontract.ComplexContract {
	cc := new(complexcontract.ComplexContract)

	cc.SetTransactionContextHandler(new(utils.CustomTransactionContext))
	cc.SetBeforeTransaction(utils.GetWorldState)
	cc.SetUnknownTransaction(utils.UnknownTransactionHandler)

	return cc
}

type suiteContext struct {
	lastResponse peer.Response
	stub         *shimtest.MockStub
	chaincode    contractapi.ContractChaincode
}

func (sc *suiteContext) createChaincode(name string) error {

	if _, ok := contractsMap[name]; !ok {
		return fmt.Errorf("Invalid contract name %s", name)
	}

	chaincode, err := contractapi.CreateNewChaincode(contractsMap[name])

	if err != nil {
		return fmt.Errorf("expected to get nil for error on create chaincode but got " + err.Error())
	}

	sc.chaincode = chaincode
	sc.stub = shimtest.NewMockStub(name, &sc.chaincode)

	return nil
}

func (sc *suiteContext) createChaincodeAndInit(name string) error {
	err := sc.createChaincode(name)

	if err != nil {
		return err
	}

	return sc.testInitialise()
}

func (sc *suiteContext) testInitialise() error {
	txID := strconv.Itoa(rand.Int())

	sc.stub.MockTransactionStart(txID)
	response := sc.stub.MockInit(txID, [][]byte{})
	sc.stub.MockTransactionEnd(txID)

	if response.GetStatus() != int32(200) {
		return fmt.Errorf("expected to get status 200 on init but got " + strconv.Itoa(int(response.GetStatus())))
	}

	return nil
}

func (sc *suiteContext) invokeChaincode(function string, argsTbl *gherkin.DataTable) error {
	txID := strconv.Itoa(rand.Int())

	argBytes := [][]byte{}
	argBytes = append(argBytes, []byte(function))

	if len(argsTbl.Rows) > 1 {
		return fmt.Errorf("expected zero or one table of args")
	}

	for _, row := range argsTbl.Rows {
		for _, cell := range row.Cells {
			argBytes = append(argBytes, []byte(cell.Value))
		}
	}

	sc.stub.MockTransactionStart(txID)
	response := sc.stub.MockInvoke(txID, argBytes)
	sc.stub.MockTransactionEnd(txID)

	sc.lastResponse = response

	println(response.GetMessage())

	return nil
}

func (sc *suiteContext) checkSuccessResponse(result string) error {
	if sc.lastResponse.GetStatus() != int32(200) {
		return fmt.Errorf("expected to get status 200 on invoke")
	}

	payload := string(sc.lastResponse.GetPayload())
	if result != "" && payload != result {
		return fmt.Errorf("expected to get payload " + result + " but got " + payload)
	}

	return nil
}

func (sc *suiteContext) checkFailedResponse(result string) error {
	if sc.lastResponse.GetStatus() == int32(200) {
		return fmt.Errorf("expected to not get status 200 on invoke")
	}

	result = fmt.Sprintf("%s", strings.Join(strings.Split(result, "\\n"), "\n"))

	message := sc.lastResponse.GetMessage()
	if result != "" && message != result {
		return fmt.Errorf("expected to get message " + result + " but got " + message)
	}

	return nil
}

func FeatureContext(s *godog.Suite) {
	sc := new(suiteContext)

	s.Step(`^I have created chaincode (?:["'](.*?)["'])$`, sc.createChaincode)
	s.Step(`^I have created and initialised chaincode (?:["'](.*?)["'])$`, sc.createChaincodeAndInit)
	s.Step(`^I should be able to initialise the chaincode`, sc.testInitialise)
	s.Step(`^I submit the (?:"(.*?)") transaction$`, sc.invokeChaincode)
	s.Step(`^I should receive a successful response\s?(?:(?:["'](.*?)["'])?)$`, sc.checkSuccessResponse)
	s.Step(`^I should receive an unsuccessful response\s?(?:(?:["'](.*?)["'])?)$`, sc.checkFailedResponse)
}
