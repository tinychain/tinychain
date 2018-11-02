package db

import (
	"strconv"

	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core/types"
	"github.com/op/go-logging"
)

/*
	** Hash is block header hash


	"LastHeader" => the latest block header
	"LastBlock" => the latest block
	"WorldState" => the latest world state root hash

	"h" + block height + "n" => block hash
	"h" + block height + block hash => header
	"H" + block hash => block height
	"b" + block height + block hash => block
	"r" + txHash => block receipt
	"l" + txHash => transaction meta data {hash,height,txIndex}
*/

var log *logging.Logger // package-level logger

const (
	KeyLastHeader = "LastHeader"
	KeyLastBlock  = "LastBlock"
)

func init() {
	log = logging.MustGetLogger("tinydb")
}

// TinyDB stores and manages blockchain data
type TinyDB struct {
	db Database
}

func NewTinyDB(db Database) *TinyDB {
	return &TinyDB{db}
}

func (tdb *TinyDB) LDB() Database {
	return tdb.db
}

func (tdb *TinyDB) GetLastBlock() (common.Hash, error) {
	data, err := tdb.db.Get([]byte(KeyLastBlock))
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(data), nil
}

func (tdb *TinyDB) PutLastBlock(batch Batch, hash common.Hash, sync, flush bool) error {
	if err := batch.Put([]byte(KeyLastBlock), hash.Bytes()); err != nil {
		return err
	}
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GetLastHeader() (common.Hash, error) {
	data, err := tdb.db.Get([]byte(KeyLastHeader))
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(data), nil
}

func (tdb *TinyDB) PutLastHeader(batch Batch, hash common.Hash, sync, flush bool) error {
	if err := batch.Put([]byte(KeyLastHeader), hash.Bytes()); err != nil {
		return err
	}
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GetHash(height uint64) (common.Hash, error) {
	var hash common.Hash
	data, err := tdb.db.Get([]byte("h" + strconv.FormatUint(height, 10) + "n"))
	if err != nil {
		return hash, err
	}
	hash = common.DecodeHash(data)
	return hash, nil
}

func (tdb *TinyDB) PutHash(batch Batch, height uint64, hash common.Hash, sync, flush bool) error {
	if err := batch.Put([]byte("h"+strconv.FormatUint(height, 10)+"n"), hash[:]); err != nil {
		return err
	}
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GetHeader(height uint64, hash common.Hash) (*types.Header, error) {
	data, err := tdb.db.Get([]byte("h" + strconv.FormatUint(height, 10) + hash.String()))
	if err != nil {
		return nil, err
	}
	header := types.Header{}
	header.Desrialize(data)
	return &header, nil
}

func (tdb *TinyDB) PutHeader(batch Batch, header *types.Header, sync, flush bool) error {
	data, _ := header.Serialize()
	if err := batch.Put([]byte("h"+strconv.FormatUint(header.Height, 10)+header.Hash().String()), data); err != nil {
		return err
	}
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GetHeight(hash common.Hash) (uint64, error) {
	data, err := tdb.db.Get([]byte("H" + hash.String()))
	if err != nil {
		return 0, err
	}
	return common.Bytes2Uint(data), nil
}

func (tdb *TinyDB) PutHeight(batch Batch, hash common.Hash, height uint64, sync, flush bool) error {
	if err := batch.Put([]byte("H"+hash.String()), common.Uint2Bytes(height)); err != nil {
		return err
	}
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GetBlock(height uint64, hash common.Hash) (*types.Block, error) {
	data, err := tdb.db.Get([]byte("b" + strconv.FormatUint(height, 10) + hash.String()))
	if err != nil {
		return nil, err
	}
	block := types.Block{}
	block.Deserialize(data)
	return &block, nil
}

func (tdb *TinyDB) PutBlock(batch Batch, block *types.Block, sync, flush bool) error {
	height := block.Height()
	hash := block.Hash()
	data, _ := block.Serialize()
	if err := batch.Put([]byte("b"+strconv.FormatUint(height, 10)+hash.String()), data); err != nil {
		return err
	}
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GetReceipt(txHash common.Hash) (*types.Receipt, error) {
	data, err := tdb.db.Get([]byte("r" + txHash.String()))
	if err != nil {
		return nil, err
	}
	var receipt types.Receipt
	err = receipt.Deserialize(data)
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

func (tdb *TinyDB) PutReceipt(batch Batch, txHash common.Hash, receipt *types.Receipt, sync, flush bool) error {
	data, err := receipt.Serialize()
	if err != nil {
		return err
	}
	if err := batch.Put([]byte("r"+txHash.String()), data); err != nil {
		return err
	}

	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}

func (tdb *TinyDB) GetTxMeta(txHash common.Hash) (*types.TxMeta, error) {
	data, err := tdb.db.Get([]byte("l" + txHash.String()))
	if err != nil {
		log.Errorf("Cannot find txMeta with txHash %s", txHash.Hex())
		return nil, err
	}
	txMeta := &types.TxMeta{}
	txMeta.Deserialize(data)
	return txMeta, nil
}

// PutTxMetas put transactions' meta to db in batch
func (tdb *TinyDB) PutTxMetas(batch Batch, txs types.Transactions, hash common.Hash, height uint64, sync, flush bool) error {
	for i, tx := range txs {
		txMeta := &types.TxMeta{
			Hash:    hash,
			Height:  height,
			TxIndex: uint64(i),
		}
		data, _ := txMeta.Serialize()
		if err := batch.Put([]byte("l"+tx.Hash().String()), data); err != nil {
			return err
		}
	}
	if flush {
		if sync {
			return batch.Write()
		} else {
			go batch.Write()
		}
	}
	return nil
}
