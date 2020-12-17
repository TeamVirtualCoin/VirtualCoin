package main

import(
	"net/http"
	"os"
	"fmt"
	"log"
	"encoding/json"
	"time"
	"strconv"
	"github.com/gorilla/mux"
	"github.com/asdine/storm"
	"github.com/robertkrimen/otto"
	"github.com/teamvirtualcoin/libhashx"
)
var port = os.Getenv("PORT")
var txdb,err7 = storm.Open("virtualcoin.txdb")
var coindb,err8 = storm.Open("virtualcoin.coindb")
var mnemonics = []string{"dog","cat","tiger","lion","elephant","crocodile","rabbit","rat","chicken",
	"cheetah","puma","alligator","cow","buffalo","dinosaur","cockroach","trees"}
var hashx = libhashx.LibHashX{Mnemonic : mnemonics,Length : 20}

type Transaction struct {
	Txid int `storm:"id,increment" json:"txid"`
	Amount float64 `json:"amount"`
	Sender string `json:"sender"`
	Receiver string `json:"receiver"`
	Timestamp float64 `storm:"index" json:"timestamp"`
	Txtype string `json:"txtype"`
	Code string `json:"code"`
}

type jsonTx struct {
	privateKey string 
	receiver string
	amount float64
}

type jsonSupply struct {
	privateKey string
	amount float64
	txtype string
}

type jsonSc struct {
	txid int
	privateKey string
	code string
}

type jsonCall struct {
	txid int
	privateKey string
	call string
	maxAllowance float64
}
type Wallet struct {
	mnemonic string
	privateKey string
	publicKey string
}

type Libhttp struct {
	Get func(string,interface{}) interface{}
	Post func(string,interface{}) interface{}
}

type Core struct {
	Set func(interface{},interface{})
	SetUser func(string,interface{},interface{})
	Get func(interface{}) interface{}
	GetUser func(string,interface{}) interface{}
	SendTx func(string,float64,string) interface{}
	CallContract func(int,string,string,float64) interface{}
	LibHttp Libhttp
}

func CreateWallet() []string {
	key := hashx.GenPriv()
	publicKey := hashx.GenPub(key[0])
	return []string{key[1],key[0],publicKey}
}

func MnemonicToPrivate(mnemonic string) string {
	return libhashx.Hash(mnemonic)
}

func PrivateToPublic(privateKey string) string {
	return hashx.GenPub(privateKey)
}

func GetBal(publicKey string) float64 {
	out := 0
	out2 := 0
	var this float64
	this = 0
	var inputs []Transaction
	var outputs []Transaction
	errre := txdb.Find("Receiver",publicKey,&inputs)
	errse := txdb.Find("Sender",publicKey,&outputs)
	if errre != nil {
		out += 1
	}
	if errse != nil {
		out2 += 1
	}
	var ia float64
	var oa float64
	for _,i := range inputs {
		ia += i.Amount
	}
	for _,k := range outputs {
		oa += k.Amount
	}
	if out != 1 {
		this += ia
	}
	if out2 != 1 {
		this -= oa
	}
	if oa > ia {return 0}
	return this
}

func TotalSupply() float64 {
	out := 0
    out2 := 0
    var this float64
    this = 0
    var inputs []Transaction
    var outputs []Transaction
    errre := txdb.Find("Txtype","mint",&inputs)
    errse := txdb.Find("Txtype","burn",&outputs)
    if errre != nil {
        out += 1
    }
    if errse != nil {
        out2 += 1
    }
    var ia float64
    var oa float64
    for _,i := range inputs {
        ia += i.Amount
    }
    for _,k := range outputs {
        oa -= k.Amount
    }
    if out != 1 {
        this += ia
    }
    if out2 != 1 {
        this -= oa
    }
    if oa > ia {return 0}
    return this
}

func Mint(privateKey string,amount float64) interface{} {
	pubKey := hashx.GenPub(privateKey)
	if pubKey != "3ef416197407a4324f961bd6f2dad2e003f7c8531ee261af2c5ca9e382b11483" {
		return false
	}
	thistx := Transaction{
		Amount : amount,
		Sender : "supply",
		Receiver : pubKey,
		Timestamp : float64(int(time.Now().UnixNano()) / 1e6),
		Txtype : "mint",
	}
	errmint := txdb.Save(&thistx)
	if errmint != nil {
		return false
	}
	return "true"
}

func Burn(privateKey string,amount float64) interface{} {
	pubKey := hashx.GenPub(privateKey)
    if pubKey != "3ef416197407a4324f961bd6f2dad2e003f7c8531ee261af2c5ca9e382b11483" {
        return false
    }
    if GetBal(pubKey) < amount {
    	return false
    }
    thistx := Transaction{
        Amount : amount,
        Sender : pubKey,
        Receiver : "supply",
        Timestamp : float64(int(time.Now().UnixNano()) / 1e6),
        Txtype : "burn",
    }
    errburn := txdb.Save(&thistx)
    if errburn != nil {
        return false
    }
    return "true"
}

func SendTx(privateKey string,amount float64,receiver string) string {
	pubKey := hashx.GenPub(privateKey)
	bal := GetBal(pubKey)
	if bal - amount <= 0 {
		return "error"
	}
	if amount <= 0 {
		return "error"
	}
	if len(privateKey) != 64 && len(receiver) != 64 {
		return "error"
	}
	thistx := Transaction{
		Amount : amount,
		Sender : pubKey,
		Receiver : receiver,
		Timestamp : float64(int(time.Now().UnixNano()) / 1e6),
		Txtype : "normal",
	}
	ak,zv := json.Marshal(thistx)
	if zv != nil {
		return "error"
	}
	errsatx := txdb.Save(&thistx)
	if errsatx != nil {
		return "error"
	}
	return string(ak)
}

func EstimateContractFuel(code string) float64 {
	return float64(len(code)) * 0.0001
}

func SendContract(privateKey string,code string) string {
	amount := EstimateContractFuel(code)
	pubKey := hashx.GenPub(privateKey)
	bal := GetBal(pubKey)
	if bal - amount <= 0 {
		return "error"
	}
	if len(privateKey) != 64 && len(code) <= 0 {
		return "error"
	}
	thistx := Transaction{
		Amount : amount,
		Sender : pubKey,
		Receiver : "3ef416197407a4324f961bd6f2dad2e003f7c8531ee261af2c5ca9e382b11483",
		Timestamp : float64(int(time.Now().UnixNano()) / 1e6),
		Txtype : "contract",
		Code : code,
	}
	ak,zv := json.Marshal(thistx)
	if zv != nil {
		return "error"
	}
	errsatx := txdb.Save(&thistx)
	if errsatx != nil {
		return "error"
	}
	return string(ak)
}

func IsContract(txid int) bool {
	var tx Transaction
	err := json.Unmarshal([]byte(GetTxById(txid)),&tx)
	if err != nil {
		return false
	}
	if tx.Txtype == "contract" {
		return true
	}
	return false
}

func CallFunc(txid int,privateKey string,call string,maxAllowance float64,value chan string) {
	pubKey := hashx.GenPub(privateKey)
	if GetBal(pubKey) < maxAllowance {
		value <- "error"
		return
	}
	if IsContract(txid) == false {
		value <- "error"
		return
	}
	var tx Transaction
	err := json.Unmarshal([]byte(GetTxById(txid)),&tx)
	if err != nil {
		value <- "error"
		return
	}
	id := strconv.Itoa(txid)
	core := Core{
		Set : func (variable interface{},value interface{}) {
		    coindb.Set(id,variable,value)
		},
		SetUser : func (user string,variable interface{},value interface{}) {
		    coindb.Set(id + user,variable,value)
		},
		Get : func (variable interface{}) interface{} {
			var value interface{}
			coindb.Get(id,variable,&value)
			return value
		},
		GetUser : func (user string,variable interface{}) interface{} {
			var value interface{}
			coindb.Get(id + user,variable,&value)
			return value
		},
		SendTx : func (sendertx string,amount float64,receiver string) interface{} {
			var sendtxer string
			if sendertx == pubKey {
				sendtxer = pubKey
				if amount > maxAllowance {
					return false
				}
			} else {
				sendtxer := hashx.GenPub(sendertx)
				if GetBal(sendtxer) < amount {
					return false
				}
			}
			thistx := Transaction{
				Amount : amount,
				Sender : sendtxer,
				Receiver : receiver,
				Timestamp : float64(int(time.Now().UnixNano()) / 1e6),
				Txtype : "normal",
			}
			ak,zv := json.Marshal(thistx)
			if zv != nil {
				return false
			}
			errsatx := txdb.Save(&thistx)
			if errsatx != nil {
				return false
			}
			return string(ak)
		},
		
		CallContract : func (txidc int,privateKeyc string,callc string,maxAllowancec float64) interface{} {
			valuec := make(chan string)
			go CallFunc(txidc,privateKeyc,callc,maxAllowancec,valuec)
			select {
				case call := <-valuec:
					if call == "error" {
						return false
					}
					return call
				case <-time.After(20 * time.Second):
					return false
			}
		},
	}
	vm := otto.New()
	vm.Run(tx.Code)
	vm.Set("deployer",tx.Sender)
	vm.Set("sender",pubKey)
	vm.Set("core",core)
	str,err3 := vm.Run(call)
	if err3 != nil {
		value <- "error"
		return
	}
	str2,err4 := str.ToString()
	if err4 != nil {
		value <- "error"
		return
	}
	value <- str2
}

func GetTxById(id int) string {
	var txbyid Transaction
	err := txdb.Find("Txid",id,&txbyid)
	txd, erken := json.Marshal(txbyid)
	if err != nil && erken != nil{
		return "error"
	}
	return string(txd)
}

func TotalReceivedTx(publicKey string) string {
	var inputs []Transaction
	err := txdb.Find("Receiver",publicKey,&inputs)
	jzon, erzkene := json.Marshal(inputs)
	if err != nil {
		return string(jzon)
	}
	if erzkene != nil {
		return "error"
	}
	return string(jzon)
}

func TotalSentTx(publicKey string) string {
	var outputs []Transaction
	err := txdb.Find("Sender",publicKey,&outputs)
	jzon, erzkene := json.Marshal(outputs)
	if err != nil {
		return string(jzon)
	}
	if erzkene != nil {
		return "error"
	}
	return string(jzon)
}

func main() {
	if port == "" {
		port = "3000"
	}
	if err7 != nil && err8 != nil {
		log.Fatal("Cant Open Database")
	}
	r := mux.NewRouter()
	r.HandleFunc("/createwallet",func(res http.ResponseWriter,req *http.Request){
		nwallet := CreateWallet()
		wallet := map[string]string{"mnemonic" : nwallet[0],"privateKey" : nwallet[1],"publicKey" : nwallet[2]}
		jxon, errvf := json.Marshal(wallet)
		if errvf != nil {
			http.Error(res,"Error While Creating A Wallet, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res,string(jxon))
	}).Methods("GET")
	r.HandleFunc("/sendtx",func(res http.ResponseWriter,req *http.Request) {
		var tx jsonTx
		errkn := json.NewDecoder(req.Body).Decode(&tx)
		if errkn != nil {
			http.Error(res,"Error While Parsing Your Transaction Body, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		oktx := SendTx(tx.privateKey,tx.amount,tx.receiver)
		if oktx == "error" {
			http.Error(res,"Error While Sending Your Transaction, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res,oktx)
	}).Methods("POST")
	r.HandleFunc("/editsupply",func(res http.ResponseWriter,req *http.Request) {
        var tx jsonSupply
        errkn := json.NewDecoder(req.Body).Decode(&tx)
        if errkn != nil {
            http.Error(res,"Error While Parsing Your Transaction Body, Retry Or Contact The Team",http.StatusBadRequest)
            return
        }
        var oktx interface{}
        if tx.txtype == "mint" {
        	oktx = Mint(tx.privateKey,tx.amount)
        } else if tx.txtype == "burn" {
        	oktx = Burn(tx.privateKey,tx.amount)
        }
        if oktx == false || oktx == nil {
            http.Error(res,"Error While Sending Your Transaction, Retry Or Contact The Team",http.StatusBadRequest)
            return
        }
        res.WriteHeader(http.StatusOK)
        fmt.Fprintf(res,"true")
    }).Methods("POST")
	r.HandleFunc("/sendcontract",func(res http.ResponseWriter,req *http.Request) {
		var tx jsonSc
		errkn := json.NewDecoder(req.Body).Decode(&tx)
		if errkn != nil {
			http.Error(res,"Error While Parsing Your Contract, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		oktx := SendContract(tx.privateKey,tx.code)
		if oktx == "error" {
			http.Error(res,"Error While Sending Your Contract, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res,oktx)
	}).Methods("POST")
	r.HandleFunc("/callcontract",func(res http.ResponseWriter,req *http.Request) {
		var fc jsonCall
		errkn := json.NewDecoder(req.Body).Decode(&fc)
		if errkn!= nil {
			http.Error(res,"Error While Parsing Your Call, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		status := make(chan string)
		go CallFunc(fc.txid,fc.privateKey,fc.call,fc.maxAllowance,status)
		select {
			case call := <-status:
				if call == "error" {
					http.Error(res,"Error While Executing Your Call, Retry Or Contact The Team",http.StatusBadRequest)
					return
				}
				res.WriteHeader(http.StatusOK)
				fmt.Fprintf(res,call)
				return
			case <-time.After(20 * time.Second):
				http.Error(res,"Function Takes Too Much Time, Timeout, Retry Or Contact The Token Team/Team",http.StatusBadRequest)
				return
		}
	}).Methods("POST")
	r.HandleFunc("/iscontract/{txid}",func(res http.ResponseWriter,req *http.Request) {
		tx := mux.Vars(req)["txid"]
		id,err := strconv.Atoi(tx)
		var a string
		if IsContract(id) {
			a = "true"
		} else {
			a = "false"
		}
		if err != nil {
			http.Error(res,"Please Only Include Integers/Numbers, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res,a)
	})
	r.HandleFunc("/contractfuel/{code}",func(res http.ResponseWriter,req *http.Request) {
		code := mux.Vars(req)["code"]
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res,fmt.Sprintf("%f",EstimateContractFuel(code)))
	}).Methods("GET")
	r.HandleFunc("/gettx/{id}",func(res http.ResponseWriter,req *http.Request) {
		id := mux.Vars(req)["id"]
		a,j := strconv.Atoi(id)
		if j != nil {
			http.Error(res,"Please Only Include Integers/Numbers, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		txc := GetTxById(a)
		if txc == "error" {
			http.Error(res,"This Transaction Doesnt Exist/Error, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res,txc)
	}).Methods("GET")
	r.HandleFunc("/balance/{address}",func(res http.ResponseWriter,req *http.Request) {
		address := mux.Vars(req)["address"]
		fmt.Fprintf(res,fmt.Sprintf("%f",GetBal(address)))
	}).Methods("GET")
	r.HandleFunc("/totalsupply",func(res http.ResponseWriter,req *http.Request) {
        fmt.Fprintf(res,fmt.Sprintf("%f",TotalSupply()))
    }).Methods("GET")
	r.HandleFunc("/receivedtx/{address}",func(res http.ResponseWriter,req *http.Request) {
		address := mux.Vars(req)["address"]
		txs := TotalReceivedTx(address)
		if txs == "error" {
			http.Error(res,"Error Cant Check Received Txs, Retry Or Contact The Team",http.StatusBadRequest)
			return
		}
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res,string(txs))
	}).Methods("GET")
	r.HandleFunc("/senttx/{address}",func(res http.ResponseWriter,req *http.Request) {
	    address := mux.Vars(req)["address"]
	    txs := TotalSentTx(address)
	    if txs == "error" {
	        http.Error(res,"Error Cant Check Sent Txs, Retry Or Contact The Team",http.StatusBadRequest)
	        return
	    }
	    res.WriteHeader(http.StatusOK)
	    fmt.Fprintf(res,(string(txs)))
	}).Methods("GET")
	fmt.Println("VirtualCoin - The First VirtualCurrency")
	fmt.Println("Listening On Port " + port)
	fmt.Println("Version : v0.0.7 BETA")
	fmt.Println("By QuazoNetwork")
	http.ListenAndServe(":" + port,r)
}
