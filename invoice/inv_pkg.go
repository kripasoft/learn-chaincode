package main

import (
	"fmt"
	"strconv"
	"encoding/json"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"errors"
)

//==============================================================================================================================
//	Structure Definitions
//==============================================================================================================================
//	Chaincode - A blank struct for use with Shim (A HyperLedger included go file used for get/put state
//				and other HyperLedger functions)
//==============================================================================================================================
type  SimpleChaincode struct {
}		

//==============================================================================================================================
//	Account - Defines the structure for an account object. JSON on right tells it what JSON fields to map to
//	Invoice - Defines the structure for an invoice object. JSON on right tells it what JSON fields to map to
//			  that element when reading a JSON object into the struct e.g. JSON currency -> Struct Currency
//==============================================================================================================================
/* type Account struct{
	AccountNo string `json:"accountno"`	
	LegalEntity string `json:"legalentity"`
	Currency string `json:"currency"`				
	Balance string `json:"balance"`
}*/
type Invoice struct{
	InvoiceNo string `json:"invoiceno"`	
	LegalEntity string `json:"legalentity"`
	Currency string `json:"currency"`				
	Balance string `json:"balance"`
}

var invoiceIndexStr = "_invoiceindex"	  // Define an index varibale to track all the invoices stored in the world state

// ============================================================================================================================
//  Main - main - Starts up the chaincode
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// ============================================================================================================================
// Init Function - Called when the user deploys the chaincode
// ============================================================================================================================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting a single integer")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for testing the blockchain network")
	}

	// Write the state to the ledger, test the network
	err = stub.PutState("test_key", []byte(strconv.Itoa(Aval)))	
	if err != nil {
		return nil, err
	}
	
	var empty []string
	jsonAsBytes, _ := json.Marshal(empty)								//marshal an emtpy array of strings to clear the invoice index
	err = stub.PutState(invoiceIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}
	
	return nil, nil
}

// ============================================================================================================================
// Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//		    initial arguments passed to other things for use in the called function.
// ============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	// Handle different functions
	if function == "init" {										//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "delete" {									
		return t.Delete(stub, args)												
	} else if function == "write" {									
		return t.Write(stub, args)
	} else if function == "init_invoice" {									
		return t.init_invoice(stub, args)
	} else if function == "transfer_balance" {									
		return t.transfer_balance(stub, args)										
	}

	return nil, errors.New("Received unknown function invocation: " + function)
}

// ============================================================================================================================
//	Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//  		initial arguments passed are passed on to the called function.
// ============================================================================================================================
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	if function == "read" {												
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function)						//error

	return nil, errors.New("Received unknown function query " + function)
}

// ============================================================================================================================
// Read - read a variable from chaincode world state
// ============================================================================================================================
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name)	
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil												
}

// ============================================================================================================================
// Delete - remove a key/value pair from the world state
// ============================================================================================================================
func (t *SimpleChaincode) Delete(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}
	
	name := args[0]
	err := stub.DelState(name)													//remove the key from chaincode state
	if err != nil {
		return nil, errors.New("Failed to delete state")
	}

	//get the invoice index
	invoicesAsBytes, err := stub.GetState(invoiceIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get invoice index")
	}
	var invoiceIndex []string
	json.Unmarshal(invoicesAsBytes, &invoiceIndex)						
	
	//remove invoice from index
	for i,val := range invoiceIndex{
		if val == name{															//find the correct invoice
			invoiceIndex = append(invoiceIndex[:i], invoiceIndex[i+1:]...)			//remove it
			break
		}
	}
	jsonAsBytes, _ := json.Marshal(invoiceIndex)									//save the new index
	err = stub.PutState(invoiceIndexStr, jsonAsBytes)
	return nil, nil
}

// ============================================================================================================================
// Write - directly write a variable into chaincode world state
// ============================================================================================================================
func (t *SimpleChaincode) Write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, value string 
	var err error

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
	}

	name = args[0]														
	value = args[1]
	err = stub.PutState(name, []byte(value))					
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// Init invoice - create a new invoice, store into chaincode world state, and then append the invoice index
// ============================================================================================================================
func (t *SimpleChaincode) init_invoice(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//       0        1      2      3
	// "invoiceNo", "bob", "USD", "3500"

	if len(args) != 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	//input sanitation
	fmt.Println("- start init acount")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("3rd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return nil, errors.New("3rd argument must be a non-empty string")
	}

	invoiceNo := args[0]

	amount := strings.ToLower(args[1])

	currency := args[2]

	ammount, err := strconv.ParseFloat(args[3],64)
	if err != nil {
		return nil, errors.New("4rd argument must be a numeric string")
	}

	//check if invoice already exists
	invoiceAsBytes, err := stub.GetState(invoiceNo)
	if err != nil {
		return nil, errors.New("Failed to get invoice number")
	}
	res := Invoice{}
	json.Unmarshal(invoiceAsBytes, &res)
	//if res.AccountNo == invoiceNo{
	if res.InvoiceNo == invoiceNo{
		return nil, errors.New("This invoice arleady exists")			
	}
	amountStr := strconv.FormatFloat(ammount, 'E', -1, 64)

	//build the invoice json string 
	str := `{"invoiceno": "` + invoiceNo + `", "amount": "` + amount + `", "currency": "` + currency + `", "balance": "` + amountStr + `"}`
	err = stub.PutState(invoiceNo, []byte(str))							
	if err != nil {
		return nil, err
	}
		
	//get the invoice index
	invoicesAsBytes, err := stub.GetState(invoiceIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get invoice index")
	}
	var invoiceIndex []string
	json.Unmarshal(invoicesAsBytes, &invoiceIndex)							
	
	//append the index 
	invoiceIndex = append(invoiceIndex, invoiceNo)	
	jsonAsBytes, _ := json.Marshal(invoiceIndex)
	err = stub.PutState(invoiceIndexStr, jsonAsBytes)						

	return nil, nil
}

// ============================================================================================================================
// Transfer Balance - Create a transaction between two invoices, transfer a certain amount of balance
// ============================================================================================================================
func (t *SimpleChaincode) transfer_balance(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	
	//       0           1         2
	// "invoiceA", "invoiceB", "100.20"

	var err error
	var newAmountA, newAmountB float64

	if len(args) < 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	amount,err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return nil, errors.New("3rd argument must be a numeric string")
	}

	invoiceAAsBytes, err := stub.GetState(args[0])
	if err != nil {
		return nil, errors.New("Failed to get the first invoice")
	}
	resA := Invoice{}
	json.Unmarshal(invoiceAAsBytes, &resA)								
	
	invoiceBAsBytes, err := stub.GetState(args[1])
	if err != nil {
		return nil, errors.New("Failed to get the second invoice")
	}
	resB := Invoice{}
	json.Unmarshal(invoiceBAsBytes, &resB)											
	
	BalanceA,err := strconv.ParseFloat(resA.Balance, 64)
	if err != nil {
		return nil, err
	}
	BalanceB,err := strconv.ParseFloat(resB.Balance, 64)
	if err != nil {
		return nil, err
	}

	//Check if invoiceA has enough balance to transact or not
	if (BalanceA - amount) < 0 {
		return nil, errors.New(args[0] + " doesn't have enough balance to complete transaction")
	}

	newAmountA = BalanceA - amount
	newAmountB =  BalanceB + amount
	newAmountStrA := strconv.FormatFloat(newAmountA, 'E', -1, 64)
	newAmountStrB := strconv.FormatFloat(newAmountB, 'E', -1, 64)

	resA.Balance = newAmountStrA
	resB.Balance = newAmountStrB

	jsonAAsBytes, _ := json.Marshal(resA)
	err = stub.PutState(args[0], jsonAAsBytes)								
	if err != nil {
		return nil, err
	}

	jsonBAsBytes, _ := json.Marshal(resB)
	err = stub.PutState(args[1], jsonBAsBytes)								
	if err != nil {
		return nil, err
	}
	
	return nil, nil
}
