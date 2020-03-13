package bsc

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"blockbook/bchain/coins/utils"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/chaincfg"
)

// magic numbers
const (
	MainnetMagic wire.BitcoinNet = 0xf9beb4d9
	TestnetMagic wire.BitcoinNet = 0x0b110907
)

// chain parameters
var (
	MainNetParams chaincfg.Params
	TestNetParams chaincfg.Params
)

func init() {
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Net = MainnetMagic
	//MainNetParams.PubKeyHashAddrID = []byte{58}
	//MainNetParams.ScriptHashAddrID = []byte{50}
	//MainNetParams.Bech32HRPSegwit = "qc"

	TestNetParams = chaincfg.TestNet3Params
	TestNetParams.Net = TestnetMagic
	//TestNetParams.PubKeyHashAddrID = []byte{120}
	//TestNetParams.ScriptHashAddrID = []byte{110}
	//TestNetParams.Bech32HRPSegwit = "tq"
}

// BSCParser handle
type BscParser struct {
	*btc.BitcoinParser
}

// NewBscParser returns new DashParser instance
func NewBscParser(params *chaincfg.Params, c *btc.Configuration) *BscParser {
	return &BscParser{
		BitcoinParser: btc.NewBitcoinParser(params, c),
	}
}

// GetChainParams contains network parameters for the main Bsc network,
// the regression test Bsc network, the test Bsc network and
// the simulation test Bsc network, in this order
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err == nil {
			err = chaincfg.Register(&TestNetParams)
		}
		if err != nil {
			panic(err)
		}
	}
	switch chain {
	case "test":
		return &TestNetParams
	default:
		return &MainNetParams
	}
}

func parseBlockHeader(r io.Reader) (*wire.BlockHeader, error) {
	h := &wire.BlockHeader{}
	err := h.Deserialize(r)
	if err != nil {
		return nil, err
	}

	// hash_state_root 32
	// hash_utxo_root 32
	// hash_prevout_stake 32
	// hash_prevout_n 4
	buf := make([]byte, 100)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	sigLength, err := wire.ReadVarInt(r, 0)
	if err != nil {
		return nil, err
	}
	sigBuf := make([]byte, sigLength)
	_, err = io.ReadFull(r, sigBuf)
	if err != nil {
		return nil, err
	}

	return h, err
}

func (p *BscParser) ParseBlock(b []byte) (*bchain.Block, error) {

	height := binary.BigEndian.Uint32(b[:8])
	if height < 35000 {
		return p.BitcoinParser.ParseBlock(b[8:])
	}

	r := bytes.NewReader(b[8:])
	w := wire.MsgBlock{}

	h, err := parseBlockHeader(r)
	if err != nil {
		return nil, err
	}

	err = utils.DecodeTransactions(r, 0, wire.WitnessEncoding, &w)
	if err != nil {
		return nil, err
	}

	txs := make([]bchain.Tx, len(w.Transactions))
	for ti, t := range w.Transactions {
		txs[ti] = p.TxFromMsgTx(t, false)
	}

	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Size: len(b),
			Time: h.Timestamp.Unix(),
		},
		Txs: txs,
	}, nil
}

// ParseTxFromJson parses JSON message containing transaction and returns Tx struct
func (p *BscParser) ParseTxFromJson(msg json.RawMessage) (*bchain.Tx, error) {
	var tx bchain.Tx
	err := json.Unmarshal(msg, &tx)
	if err != nil {
		return nil, err
	}

	for i := range tx.Vout {
		vout := &tx.Vout[i]
		// convert vout.JsonValue to big.Int and clear it, it is only temporary value used for unmarshal
		vout.ValueSat, err = p.AmountToBigInt(vout.JsonValue)
		if err != nil {
			return nil, err
		}
		vout.JsonValue = ""

		if vout.ScriptPubKey.Addresses == nil {
			vout.ScriptPubKey.Addresses = []string{}
		}
	}

	return &tx, nil
}
