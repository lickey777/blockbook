package bsc

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"encoding/binary"
	"encoding/json"
	"math/big"

	"github.com/golang/glog"
	"github.com/juju/errors"
)

// BscRPC is an interface to JSON-RPC bitcoind service.
type BscRPC struct {
	*btc.BitcoinRPC
	minFeeRate *big.Int // satoshi per kb
}

// NewBscRPC returns new BscRPC instance.
func NewBscRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &BscRPC{
		b.(*btc.BitcoinRPC),
		big.NewInt(400000),
	}
	s.RPCMarshaler = btc.JSONMarshalerV1{}
	s.ChainConfig.SupportsEstimateSmartFee = true

	return s, nil
}

// Initialize initializes BscRPC instance.
func (b *BscRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewBscParser(params, b.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		b.Testnet = false
		b.Network = "livenet"
	} else {
		b.Testnet = true
		b.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

// GetBlockWithoutHeader is an optimization - it does not call GetBlockHeader to get prev, next hashes
// instead it sets to header only block hash and height passed in parameters
func (b *BscRPC) GetBlockWithoutHeader(hash string, height uint32) (*bchain.Block, error) {
	data, err := b.GetBlockRaw(hash)
	if err != nil {
		return nil, err
	}

	h := make([]byte, 8)
	binary.BigEndian.PutUint32(h, height)

	block, err := b.Parser.ParseBlock(append(h, data...))

	if err != nil {
		return nil, errors.Annotatef(err, "%v %v", height, hash)
	}
	block.BlockHeader.Hash = hash
	block.BlockHeader.Height = height
	return block, nil
}

// GetBlock returns block with given hash.
func (b *BscRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" {
		hash, err = b.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}
	if !b.ParseBlocks {
		return b.GetBlockFull(hash)
	}
	// optimization
	if height > 0 {
		return b.GetBlockWithoutHeader(hash, height)
	}
	header, err := b.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}
	data, err := b.GetBlockRaw(hash)
	if err != nil {
		return nil, err
	}

	h := make([]byte, 8)
	binary.BigEndian.PutUint32(h, height)

	block, err := b.Parser.ParseBlock(append(h, data...))

	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}

	block.BlockHeader = *header
	return block, nil
}

// GetTransactionForMempool returns a transaction by the transaction ID
// It could be optimized for mempool, i.e. without block time and confirmations
func (b *BscRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
	return b.GetTransaction(txid)
}

// EstimateSmartFee returns fee estimation
func (b *BscRPC) EstimateSmartFee(blocks int, conservative bool) (big.Int, error) {
	feeRate, err := b.EstimateSmartFee(blocks, conservative)
	if err != nil {
		if b.minFeeRate.Cmp(&feeRate) == 1 {
			feeRate = *b.minFeeRate
		}
	}
	return feeRate, err
}
