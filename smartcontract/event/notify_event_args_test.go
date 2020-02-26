package event

import (
	"encoding/json"
	"github.com/DNAProject/DNA/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

// go test -count=1 -v github.com/DNAProject/DNA/smartcontract/event -run TestEventMarshal
func TestEventMarshal(t *testing.T) {
	hashhex := "7a55f97262d9f210caf84753485d9ef902bcf98a7b3e03f21fda0bd9cf4d63f4"
	addrbase58 := "AN1v1jiV4CQ1zFzgxXN2iNyQKWux7NDGs5"
	version := big.NewInt(0)

	addr, _ := common.AddressFromBase58(addrbase58)
	hash, _ := common.Uint256FromHexString(hashhex)

	states := []interface{}{
		addr,
		"success",
		version,
	}
	notify := &ExecuteNotify{
		TxHash:hash,
		State: byte(3),
		GasConsumed: uint64(1000000),
		Notify: []*NotifyEventInfo{
			&NotifyEventInfo{
				ContractAddress: addr,
				States:          states,
			},
		},
	}

	// marshal后 address类型的数据为float64:
	// {"TxHash":[244,99,77,207,217,11,218,31,242,3,62,123,138,249,188,2,249,158,93,72,83,71,248,202,16,242,217,98,114,249,85,122],
	//"State":3,"GasConsumed":1000000,
	//"Notify":[
	//{"ContractAddress":[68,120,141,169,46,210,124,244,169,223,147,70,107,34,75,227,255,192,26,43],
	//"States":[[68,120,141,169,46,210,124,244,169,223,147,70,107,34,75,227,255,192,26,43],
	//"success",0]}]}
	data, err := json.Marshal(notify)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))

	notify1 := &ExecuteNotify{}
	if err := json.Unmarshal(data, notify1); err != nil {
		t.Fatal(err)
	}
	state := notify1.Notify[0].States
	slice := state.([]interface{})

	bs := []byte{}
	for _, v := range slice[0].([]interface{}) {
		bs = append(bs, byte(v.(float64)))
	}
	addr1, _ := common.AddressParseFromBytes(bs)
	assert.Equal(t, addr.ToBase58(), addr1.ToBase58())
}
